from django.contrib import admin
from django.db import models
from django.utils.translation import gettext_lazy as _

from core.models.generic_audited_model import GenericAuditedModel
from core.models.media.media_model import Media


class MediaDocument(GenericAuditedModel):

    media = models.OneToOneField(to=Media, on_delete=models.CASCADE, editable=False)
    pages = models.PositiveIntegerField(verbose_name=_("Number of pages of the document"), null=True)
    black_and_white = models.BooleanField(
        verbose_name=_("If the document has colors (false) or is black and white (true)")
    )


class MediaDocumentAdmin(admin.ModelAdmin):
    pass


admin.site.register(MediaDocument, MediaDocumentAdmin)
