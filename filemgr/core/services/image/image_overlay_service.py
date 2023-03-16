import os

from PIL import ImageDraw, ImageFont
from PIL.Image import Image

from api.settings import BASE_DIR


class ImageOverlayService:
    BAR_HEIGHT_PX=20

    INFO_BAR_COLOR_BACKGROUND='#11141A'
    INFO_BAR_COLOR_FOREGROUND='#FEFEFE'
    INFO_BAR_GROUP='image_info_bar'
    INFO_BAR_KEY_POSITION='position'
    INFO_BAR_POSITION_TOP='TOP'
    INFO_BAR_POSITION_BOTTOM='BOTTOM'
    INFO_BAR_TEXT_LEFT='text_left'
    INFO_BAR_TEXT_CENTER='text_center'
    INFO_BAR_TEXT_RIGHT='text_right'

    INFO_BAR_KEY_BACKGROUND_COLOR='background_color'

    def __init__(self, image: Image, configuration: dict, metadata: dict = None):
        if metadata is None:
            metadata = {}

        self.font_path = os.path.join(BASE_DIR, 'public', 'fonts', 'yantramanav', 'Yantramanav-Bold.ttf')

        self.configuration = configuration
        self.image = image
        self.metadata = metadata

    def _get_configuration_key(self, group: str, key: str, default = None):
        if group not in self.configuration:
            return default

        if key not in self.configuration[group]:
            return default

        return self.configuration[group][key]

    def add_overlay(self):
        if self.INFO_BAR_GROUP in self.configuration:
            self._draw_image_info_bar(image=self.image)

    def _draw_image_info_bar(self, image: Image):
        position = self._get_configuration_key(
            group=self.INFO_BAR_GROUP,
            key=self.INFO_BAR_KEY_POSITION,
            default=self.INFO_BAR_POSITION_BOTTOM
        )

        self._draw_image_info_bar_background(position=position)

        draw = ImageDraw.Draw(self.image)
        draw.font = ImageFont.truetype(self.font_path)
        draw.text(
            xy=(10, float(self.image.height - self.BAR_HEIGHT_PX + 2)),
            text="InfoTrem",
        )

        print(self.configuration)

        return image

    def _draw_image_info_bar_background(self, position: str = INFO_BAR_POSITION_BOTTOM):
        print(position)
        bar_position = (
            0, # x0
            0 if position == self.INFO_BAR_POSITION_TOP else self.image.height - self.BAR_HEIGHT_PX, # y0
            self.image.width, # x1
            self.BAR_HEIGHT_PX if position == self.INFO_BAR_POSITION_TOP else self.image.height, # y1
        )
        print(bar_position)
        print("MAOE")

        background_color = self._get_configuration_key(
            group=self.INFO_BAR_GROUP,
            key=self.INFO_BAR_KEY_BACKGROUND_COLOR,
            default=self.INFO_BAR_COLOR_BACKGROUND
        )

        draw = ImageDraw.Draw(self.image)
        draw.rectangle(
            xy=bar_position,
            fill=background_color,
        )

    def _draw_image_info_text(self, position: str = INFO_BAR_POSITION_BOTTOM):
        left_text = self._get_configuration_key(group=self.INFO_BAR_GROUP, key=self.INFO_BAR_TEXT_LEFT, default="")
        center_text = self._get_configuration_key(group=self.INFO_BAR_GROUP, key=self.INFO_BAR_TEXT_CENTER, default="")
        right_text = self._get_configuration_key(group=self.INFO_BAR_GROUP, key=self.INFO_BAR_TEXT_RIGHT, default="")

        if not left_text and not center_text and not right_text:
            return

        draw = ImageDraw.Draw(self.image)
        draw.font = ImageFont.truetype(self.font_path)

        half_bar_px = self.BAR_HEIGHT_PX/2
        text_y = float(self.image.height - half_bar_px) if position == self.INFO_BAR_POSITION_BOTTOM else half_bar_px

        if left_text != "":
            draw.text(
                xy=(10, text_y),
                text=self._process_image_info_text(left_text),
                anchor="lm",
            )

        if center_text != "":
            draw.text(
                xy=(self.image.width / 2, text_y),
                text=self._process_image_info_text(center_text),
                anchor="mm",
            )

        if right_text != "":
            draw.text(
                xy=(self.image.width - 10, text_y),
                text=self._process_image_info_text(right_text),
                anchor="rm",
            )

    def _process_image_info_text(self, input_text: str) -> str:
        return input_text
