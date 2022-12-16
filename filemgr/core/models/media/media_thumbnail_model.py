from django.contrib import admin
from django.db import models
from django.utils.translation import gettext_lazy as _

from core.models import StorageFile
from core.models.generic_audited_model import GenericAuditedModel
from core.models.media.media_model import Media


class MediaThumbnail(GenericAuditedModel):

    media = models.ForeignKey(to=Media, on_delete=models.CASCADE, editable=False)
    size_tag = models.CharField(
        max_length=64,
        null=True,
        choices=Media.MediaSizeTag.choices,
        verbose_name=_("The size of the thumbnail.")
    )
    storage_file = models.ForeignKey(to=StorageFile, on_delete=models.CASCADE, editable=False)


class MediaThumbnailAdmin(admin.ModelAdmin):
    pass


admin.site.register(MediaThumbnail, MediaThumbnailAdmin)
