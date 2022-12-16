from django.db.models import QuerySet

from core.exceptions.model_not_found_exception import ModelNotFoundException
from core.models.media.media_model import Media
from core.models.media.media_document_model import MediaDocument
from core.models.media.media_image_model import MediaImage
from core.models.media.media_image_sized_model import MediaImageSized
from core.models.media.media_video_model import MediaVideo
from core.models.storage.storage_model import Storage
from core.models.storage.storage_file_model import StorageFile
from core.models.storage.storage_file_download_model import StorageFileDownload
from core.models.storage.storage_file_mime_type_model import StorageFileMimeType
from core.models.storage.storage_user_model import StorageUser
from core.models.user.user_model import User
from core.models.user.user_token_model import UserToken


def get_object_or_not_found_error(queryset: QuerySet, *args, **kwargs):
    try:
        return queryset.get(*args, **kwargs)
    except queryset.model.DoesNotExist:
        raise ModelNotFoundException
