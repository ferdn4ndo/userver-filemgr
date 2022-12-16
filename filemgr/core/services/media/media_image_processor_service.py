import os.path
import tempfile
from pathlib import Path

from PIL import Image, ImageFilter, ImageOps
from django.forms import model_to_dict

from core.models import StorageFile, MediaImage, Media, MediaImageSized
from core.services.logger.logger_service import get_logger
from core.services.media.media_image_service import MediaImageService
from core.services.message_broker.message_broker_service import MessageBrokerService
from core.services.storage.storage_file_service import StorageFileService


class MediaImageProcessorService:
    def __init__(self, storage_file: StorageFile):
        self.logger = get_logger(__name__)
        self.storage_file = storage_file

        self.driver = StorageFileService(storage_file=self.storage_file).load_driver()

        self.temp_file_path = self.storage_file.get_temp_file_path()
        if not os.path.isfile(self.temp_file_path):
            self.driver.download_to_path(file=self.storage_file, local_path=self.temp_file_path)

        with Image.open(self.temp_file_path) as image:
            self.original_width, self.original_height = image.size

        self.media_convert_configuration = self.storage_file.storage.media_convert_configuration

        self.media = Media(
            title=self.storage_file.name,
            type=Media.MediaType.IMAGE,
            description="",
            storage_file=self.storage_file,
        )
        self.media.save()

        self.media_image = MediaImage(
            media=self.media,
            size_tag=self.get_size_tag_from_dimensions(width=self.original_width, height=self.original_height),
            height=self.original_height,
            width=self.original_width,
            megapixels=(self.original_width * self.original_height)/1000000,
        )
        self.media_image.save()

    def process(self):
        self.logger.info(f"Processing image from storage file {self.storage_file.id}...")

        self.storage_file.status = StorageFile.FileStatus.PROCESSING
        self.storage_file.save()

        MessageBrokerService().send_message(
            topic='storage_file_processing_started',
            payload=model_to_dict(self.storage_file)
        )

        media_image_service = MediaImageService(media_image=self.media_image)
        media_image_service.update_file_metadata_from_path(image_path=self.temp_file_path)

        for image_size in self.media_convert_configuration["image_resizer"]["sizes"]:
            size_tag_width, size_tag_height = map(int, str(image_size).split("x"))
            resize_width, resize_height = self.compute_new_image_dimensions(expected_width=size_tag_width, expected_height=size_tag_height)
            if self.original_width < resize_width or self.original_height < resize_height:
                self.logger.warning(
                    f"Skipping the resize of media_image ID {self.media_image.id} for the dimensions"
                    f" {resize_width} x {resize_height} [WxH] as the original image is smaller"
                    f" ({self.original_width} x {self.original_height})"
                )
                continue

            size_tag = self.get_size_tag_from_dimensions(width=resize_width, height=resize_height)

            resized_storage_file = self.create_resized_file_resource()

            with Image.open(self.temp_file_path) as image:
                image.thumbnail((resize_width, resize_height))
                self.save_resized_image(
                    size_tag=size_tag,
                    width=resize_width,
                    height=resize_height,
                    image=image,
                    storage_file=resized_storage_file,
                )

        self.logger.info(f"Creating thumbnails for file {self.storage_file.id}...")
        self.create_thumbnails()

        self.logger.info(f"Finished processing file {self.storage_file.id}!")

        self.storage_file.status = StorageFile.FileStatus.PUBLISHED
        self.storage_file.save()

        MessageBrokerService().send_message(
            topic='storage_file_processing_finished',
            payload=model_to_dict(self.storage_file)
        )

    def compute_new_image_dimensions(self, expected_width: int, expected_height: int) -> tuple:
        size_factor = expected_width/self.original_width if self.original_height > self.original_width else expected_height/self.original_height

        return round(self.original_width * size_factor), round(self.original_height * size_factor)

    def create_resized_file_resource(self) -> StorageFile:
        resized_storage_file = self.driver.create_empty_file_resource(user=self.storage_file.created_by)
        resized_storage_file.owner = self.storage_file.owner
        resized_storage_file.exif_metadata = self.storage_file.exif_metadata
        resized_storage_file.custom_metadata = self.storage_file.custom_metadata
        resized_storage_file.name = self.storage_file.name
        resized_storage_file.original_path = self.storage_file.original_path
        resized_storage_file.visibility = self.storage_file.visibility

        return resized_storage_file

    def create_thumbnails(self):
        thumbnail_sizes = [
            {
                "tag": Media.MediaSizeTag.SIZE_THUMB_LARGE,
                "width": 960,
                "height": 720,
            },
            {
                "tag": Media.MediaSizeTag.SIZE_THUMB_MEDIUM,
                "width": 480,
                "height": 360,
            },
            {
                "tag": Media.MediaSizeTag.SIZE_THUMB_SMALL,
                "width": 240,
                "height": 180,
            },
        ]

        for thumbnail_params in thumbnail_sizes:
            self.create_thumbnail(
                size_tag=thumbnail_params["tag"],
                width=thumbnail_params["width"],
                height=thumbnail_params["height"],
            )

        print(f"FINISHED CREATING THUMBNAILS FOR FILE ID {self.storage_file.id}")

    @staticmethod
    def _get_center_box(outer_width: int, outer_height: int, inner_width: int, inner_height: int) -> tuple:
        left = round((outer_width - inner_width)/2)
        top = round((outer_height - inner_height)/2)
        right = round((outer_width + inner_width)/2)
        bottom = round((outer_height + inner_height)/2)

        return left, top, right, bottom

    def save_resized_image(
            self,
            size_tag: str,
            width: int,
            height: int,
            image: Image.Image,
            storage_file: StorageFile,
    ):
        resized_file_folder = os.path.join(tempfile.gettempdir(), "resized", size_tag)
        Path(resized_file_folder).mkdir(parents=True, exist_ok=True)
        resized_file_path = os.path.join(resized_file_folder, str(storage_file.id))
        image.save(fp=resized_file_path, format="JPEG")

        remote_path = self.driver.get_real_remote_path(file=storage_file, subfolder=f"resized/{size_tag}")

        self.driver.perform_upload_from_path(
            local_path=resized_file_path,
            remote_path=remote_path,
        )

        media_image_sized = MediaImageSized(
            media_image=self.media_image,
            storage_file=storage_file,
            size_tag=size_tag,
            height=height,
            width=width,
            megapixels=(width * height)/1000000,
        )
        media_image_sized.save()

        self.logger.info(
            f"Finished resizing file {self.storage_file.id} to size {width} x {height} (tmp path: {resized_file_path})"
        )

    def create_thumbnail(self, size_tag: str, width: int, height: int):
        thumbnail_storage_file = self.create_resized_file_resource()

        thumbnail_image = Image.new("RGB", (width, height))

        original_proportion = round(self.original_width/self.original_height, 2)
        thumbnail_proportion = round(width/height, 2)
        if thumbnail_proportion != original_proportion:
            print("ADJUSTING THUMBNAIL BACKGROUND")
            background_width, background_height = self.compute_new_image_dimensions(expected_width=width, expected_height=height)
            image = Image.open(fp=self.temp_file_path)
            image.thumbnail((background_width, background_height))

            image = image.crop(self._get_center_box(
                outer_width=background_width,
                outer_height=background_height,
                inner_width=width,
                inner_height=height,
            ))
            image = image.filter(ImageFilter.BoxBlur(radius=5))
            image = ImageOps.grayscale(image)
            thumbnail_image.paste(image)

        image = Image.open(fp=self.temp_file_path)
        image.thumbnail((width, height))
        image_width, image_height = image.size

        thumbnail_image.paste(image, self._get_center_box(
            outer_width=width,
            outer_height=height,
            inner_width=image_width,
            inner_height=image_height,
        ))

        self.save_resized_image(
            size_tag=size_tag,
            width=width,
            height=height,
            image=thumbnail_image,
            storage_file=thumbnail_storage_file,
        )

    @staticmethod
    def get_size_tag_from_dimensions(width: int, height: int):
        if width >= 8192 and height >= 5472:
            return Media.MediaSizeTag.SIZE_8K

        if width >= 4096 and height >= 2752:
            return Media.MediaSizeTag.SIZE_4K

        if width >= 3200 and height >= 2144:
            return Media.MediaSizeTag.SIZE_3K

        if width >= 2048 and height >= 1376:
            return Media.MediaSizeTag.SIZE_2K

        if width >= 1280 and height >= 864:
            return Media.MediaSizeTag.SIZE_1K

        return Media.MediaSizeTag.SIZE_VGA
