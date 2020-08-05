from __future__ import annotations

import uuid

from django.db import models
from django.db.models import Q

from filemgr.models.storage_file_model import StorageFile
from filemgr.models.storage_model import Storage
from filemgr.models.user_model import CustomUser
from filemgr.services.photo import get_image_information, get_exif_orientation, get_exif_focal_length, \
    get_exif_aperture, get_exif_flash_fired, get_exif_exposition


class StorageFileImageDimensions(models.Model):
    """
    Resource model to represent the dimensions of images for resizing/downloading
    """
    tag = models.CharField(primary_key=True, max_length=20)
    name = models.CharField(max_length=50)
    description = models.CharField(max_length=255)
    height = models.IntegerField(null=True)
    width = models.IntegerField(null=True)
    megapixels = models.FloatField(null=True)

    @staticmethod
    def from_size(height: int, width: int, approximate=False) -> StorageFileImageDimensions:
        """
        Retrieve the StorageFileImageDimensions object that corresponds to a given height and width. It may
        optionally approximate the selection (always selecting highest quality dimension that is fulfilled by
        the given size).
        :param height:
        :param width:
        :param approximate:
        :return:
        """
        try:
            return StorageFileImageDimensions.objects.get(height=height, width=width)
        except StorageFileImageDimensions.DoesNotExist:
            if not approximate:
                raise
            dimensions_list = StorageFileImageDimensions.objects.filter(
                height__lte=height,
                width__lte=width,
            ).order_by('-megapixels')
            if len(dimensions_list) == 0:
                raise
            return dimensions_list[0]


class StorageFileImage(models.Model):
    """
    Resource model represents a StorageFile of type IMAGE, with its own data
    """
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    storage_file = models.ForeignKey(to=StorageFile, on_delete=models.CASCADE, editable=False)
    raw_height = models.PositiveIntegerField(verbose_name="Height of the raw media item", null=True)
    raw_width = models.PositiveIntegerField(verbose_name="Width of the raw media item", null=True)
    dimensions = models.ForeignKey(to=StorageFileImageDimensions, on_delete=models.SET_NULL, null=True)

    def update_file_metadata(self):
        """
        Updates the models fields by reading the image metadata
        :return:
        """
        file = StorageFile.objects.get(self.storage_file)
        exif_data = get_image_information(file.get_temp_file_path())
        orientation_dict = get_exif_orientation(exif_data)
        file.metadata['raw_height'] = exif_data['IMAGE_HEIGHT']
        file.metadata['raw_width'] = exif_data['IMAGE_WIDTH']
        file.metadata['focal_length'] = get_exif_focal_length(exif_data)
        file.metadata['aperture'] = get_exif_aperture(exif_data)
        file.metadata['iso'] = exif_data['ISOSpeedRatings'] if 'ISOSpeedRatings' in exif_data else None
        file.metadata['flash_fired'] = get_exif_flash_fired(exif_data)
        file.metadata['orientation_angle'] = orientation_dict['orientation_angle'] if orientation_dict is not None else None
        file.metadata['is_flipped'] = orientation_dict['is_flipped'] if orientation_dict is not None else None
        file.metadata['exposition'] = get_exif_exposition(exif_data)
        file.metadata['datetime_taken'] = exif_data['DateTimeOriginal'] if 'DateTimeOriginal' in exif_data else None
        file.metadata['camera_manufacturer'] = exif_data['Make'] if 'Make' in exif_data else None
        file.metadata['camera_model'] = exif_data['Model'] if 'Model' in exif_data else None
        file.metadata['exif_image_height'] = exif_data['ExifImageHeight'] if 'ExifImageHeight' in exif_data else None
        file.metadata['exif_image_width'] = exif_data['ExifImageWidth'] if 'ExifImageWidth' in exif_data else None
        file.save()

    @staticmethod
    def create_storage_queryset(user: CustomUser, storage: Storage, tag: str = None):
        queryset = StorageFile.create_storage_queryset(user=user, storage=storage)
        if tag:
            queryset.filter(dimensions=tag)
        return queryset
