from django.db.models import QuerySet

from app.errors import NotFoundError

from .storage_file_download_model import StorageFileDownload, DownloadHit
from .storage_file_image_model import StorageFileImage, StorageFileImageDimensions
from .storage_file_model import StorageFile
from .storage_file_mime_type_model import StorageFileMimeType
from .storage_model import Storage
from .storage_user_model import StorageUser
from .user_model import CustomUser, UserToken


def get_object_or_404(queryset: QuerySet, *args, **kwargs):
    try:
        return queryset.get(*args, **kwargs)
    except queryset.model.DoesNotExist:
        raise NotFoundError
