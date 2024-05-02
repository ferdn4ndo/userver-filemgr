import math
import os
import re
from datetime import datetime

from PIL import ImageDraw, ImageFont
from PIL.Image import Image

from api.settings import BASE_DIR
from core.services.logger.logger_service import get_logger


class ImageOverlayService:
    BAR_HEIGHT_PX=20
    BAR_HEIGHT_RATIO=0.03

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
        self.logger = get_logger(__name__)

        if metadata is None:
            metadata = {}

        self.font_path = os.path.join(BASE_DIR, 'public', 'fonts', 'yantramanav', 'Yantramanav-Black.ttf')

        self.configuration = configuration
        self.image = image
        self.metadata = metadata

        self.text_tags_map = self._prepare_text_tags_map()

    def _prepare_text_tags_map(self):
        mapping = {}

        for key in self.metadata:
            mapping[f'metadata.{key}'] = self.metadata[key]

        mapping['YEAR'] = datetime.now().year
        mapping['MONTH'] = datetime.now().month
        mapping['DAY'] = datetime.now().day

        self.logger.debug(f"Text tags mapping: {mapping}")

        return mapping

    def _get_configuration_key(self, group: str, key: str, default = None):
        if group not in self.configuration:
            return default

        if key not in self.configuration[group]:
            return default

        return self.configuration[group][key]

    def add_overlay(self):
        if self.INFO_BAR_GROUP in self.configuration:
            self._draw_image_info_bar()

    def _compute_bar_height(self) -> int:
        return math.floor(self.image.height * self.BAR_HEIGHT_RATIO)

    def _draw_image_info_bar(self):
        position = self._get_configuration_key(
            group=self.INFO_BAR_GROUP,
            key=self.INFO_BAR_KEY_POSITION,
            default=self.INFO_BAR_POSITION_BOTTOM
        )

        self._draw_image_info_bar_background(position=position)
        self._draw_image_info_text(position=position)

        return self.image

    def _draw_image_info_bar_background(self, position: str = INFO_BAR_POSITION_BOTTOM):
        x0 = 0
        x1 = self.image.width

        if position == self.INFO_BAR_POSITION_TOP:
            y0 = 0
            y1 = self._compute_bar_height()
        else:
            y0 = self.image.height - self._compute_bar_height()
            y1 = self.image.height

        bar_position = (x0, y0, x1, y1)

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
        half_bar_px = math.ceil(0.5 * self._compute_bar_height())
        font_size = math.ceil(0.8 * self._compute_bar_height())

        self.logger.info(f"Bar height: {self._compute_bar_height()} px")
        self.logger.info(f"Half bar: {half_bar_px} px")
        self.logger.info(f"Font size: {font_size} px")

        draw.font = ImageFont.truetype(self.font_path, size=font_size)

        text_y = float(self.image.height - half_bar_px) if position == self.INFO_BAR_POSITION_BOTTOM else half_bar_px

        if left_text != "":
            draw.text(
                xy=(10, text_y),
                text=self._process_image_info_text(left_text),
                anchor="lm",
                fill=(222,222,222),
            )

        if center_text != "":
            draw.text(
                xy=(self.image.width / 2, text_y),
                text=self._process_image_info_text(center_text),
                anchor="mm",
                fill=(222,222,222),
            )

        if right_text != "":
            draw.text(
                xy=(self.image.width - 10, text_y),
                text=self._process_image_info_text(right_text),
                anchor="rm",
                fill=(222,222,222),
            )

    def _process_image_info_text(self, input_text: str) -> str:
        occurrences = re.findall(r'\{(.*?)\}', input_text)

        for occurrence in occurrences:
            if occurrence in self.text_tags_map:
                input_text = input_text.replace(f"{{{occurrence}}}", str(self.text_tags_map[occurrence]))

        return input_text
