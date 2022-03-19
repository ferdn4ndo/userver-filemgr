from __future__ import annotations

import os
import re
import tempfile
from django.core.validators import RegexValidator
from django.db import models
from django.db.models import Q
from django.utils.translation import gettext_lazy as _

from app.services.strings import generate_random_encoded_string
from app.services.translation import Messages

from .generic_model import GenericModel
from .storage_file_mime_type_model import StorageFileMimeType
from .storage_model import Storage
from .user_model import CustomUser



class StorageFile(GenericModel):
    """
    StorageFile resource model
    """

    class FileStatus(models.TextChoices):
        """
        Enum for the file status field
        """
        NOT_UPLOADED = 'NOT_UPLOADED', _('Not Uploaded')
        UPLOADING = 'UPLOADING', _('Uploading')
        QUEUED = 'QUEUED', _('Queued')
        PROCESSING = 'PROCESSING', _('Processing')
        PUBLISHED = 'PUBLISHED', _('Published')
        DELETED = 'DELETED', _('Deleted')
        ERROR = 'ERROR', _('Error')

    class FileVisibility(models.TextChoices):
        """
        Enum for the file visibility field
        """
        PUBLIC = 'PUBLIC', _('Public')
        SYSTEM = 'SYSTEM', _('System')
        USER = 'USER', _('User')

    class FileOrigin(models.TextChoices):
        """
        Enum for the file origin field. Local for raw uploaded files and web for remote downloaded ones.
        """
        LOCAL = 'LOCAL', _('Local')
        WEB = 'WEB', _('Web')
        UNKNOWN = 'UNKNOWN', _('Unknown')

    signature_key = models.CharField(max_length=64, default=generate_random_encoded_string)
    storage = models.ForeignKey(to=Storage, on_delete=models.CASCADE, editable=False)
    owner = models.ForeignKey(to=CustomUser, on_delete=models.SET_NULL, null=True, related_name="storage_file_owner")
    status = models.CharField(max_length=50, choices=FileStatus.choices, default=FileStatus.NOT_UPLOADED)
    visibility = models.CharField(max_length=50, choices=FileVisibility.choices, default=FileVisibility.SYSTEM)
    size = models.BigIntegerField(verbose_name=_("File size in bytes"), default=0)
    hash = models.CharField(max_length=255, null=True, verbose_name=_("Hash of the processed file"))
    type = models.ForeignKey(to=StorageFileMimeType, on_delete=models.SET_NULL, null=True)
    metadata = models.JSONField(default=dict, verbose_name=_("Dict with the file metadata"))
    origin = models.CharField(max_length=255, choices=FileOrigin.choices, default=FileOrigin.UNKNOWN)
    original_path = models.CharField(
        max_length=1024,
        null=True,
        verbose_name=_("Either the original URL of the file (if remotely downloaded) or the file path (if uploaded)"),
    )
    real_path = models.CharField(
        max_length=1024,
        null=True,
        validators=[
            RegexValidator(
                regex=r'^\/?([\w\.\_-]+\/)*[\w\.\_-]+$',
                message=Messages.MGS_INVALID_PATH,
                code='invalid_filepath',
                flags=re.UNICODE,
            )
        ],
        verbose_name=_("The real remote filepath of the stored file"),
    )
    virtual_path = models.CharField(
        max_length=1024,
        null=True,
        validators=[
            RegexValidator(
                regex=r'^\/?([\w\.\_-]+\/)*[\w\.\_-]+$',
                message=Messages.MGS_INVALID_PATH,
                code='invalid_filepath',
                flags=re.UNICODE,
            ),
        ],
        verbose_name=_("The virtual remote filepath of the stored file"),
    )
    available = models.BooleanField(default=True, verbose_name=_("If the file is remotely available"))
    excluded = models.BooleanField(default=False, verbose_name=_("If the file was marked as excluded in the interface"))
    created_by = models.ForeignKey(
        to=CustomUser,
        on_delete=models.SET_NULL,
        null=True,
        related_name="storage_file_creator",
    )
    created_at = models.DateTimeField(auto_now_add=True, verbose_name=_("Record creation timestamp"))
    updated_by = models.ForeignKey(
        to=CustomUser,
        on_delete=models.SET_NULL,
        null=True,
        related_name="storage_file_editor",
    )
    updated_at = models.DateTimeField(auto_now=True, verbose_name=_("Record last update timestamp"))

    def __str__(self) -> str:
        """
        Retrieves the string representation of the class
        :return:
        """
        return "<File {} #{}>".format(self.virtual_path, self.id)

    def get_temp_file_path(self) -> str:
        """
        Retrieve a local temp path for the file
        :return: the temp file path
        """
        return os.path.join(os.path.join(tempfile.gettempdir(), str(self.id)))

    def get_filename_from_virtual_path(self):
        """
        Retrieves the filename base on the file virtual path (last part of a '/' explosion)
        :return:
        """
        return str(self.virtual_path).split("/")[-1]

    def is_visible_by_user(self, user: CustomUser):
        """
        Retrieves a boolean indicating whether the file is visible to a given user or not
        :param user:
        :return:
        """
        return (
            (self.visibility == StorageFile.FileVisibility.USER and self.owner == user) or
            (self.visibility == StorageFile.FileVisibility.SYSTEM and self.owner.system_name != user.system_name) or
            (self.visibility == StorageFile.FileVisibility.PUBLIC)
        )

    @staticmethod
    def create_storage_queryset(user: CustomUser, storage: Storage, excluded: bool = False):
        queryset = StorageFile.objects.filter(storage=storage, excluded=excluded)
        queryset.filter(storage=storage).filter(
            Q(visibility=StorageFile.FileVisibility.PUBLIC) |
            (
                Q(visibility=StorageFile.FileVisibility.SYSTEM) &
                Q(owner__system_name=user.system_name)
            ) |
            (
                Q(visibility=StorageFile.FileVisibility.USER) &
                Q(owner=user)
            )
        )
        return queryset
