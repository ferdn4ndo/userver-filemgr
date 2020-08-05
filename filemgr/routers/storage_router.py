from rest_framework import routers

from filemgr.routers.route_names import RouteNames
from filemgr.views import \
    StorageFileDownloadViewSet,\
    StorageFileImageDownloadViewSet,\
    StorageFileImageViewSet,\
    StorageFileUploadView,\
    StorageFileUploadUrlView,\
    StorageFileViewSet,\
    StorageTrashViewSet,\
    StorageUserViewSet,\
    StorageViewSet

router = routers.SimpleRouter()
router.register(
    prefix=r'',
    viewset=StorageViewSet,
    basename=RouteNames.ROUTE_STORAGE
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/files',
    viewset=StorageFileViewSet,
    basename=RouteNames.ROUTE_STORAGE_FILE
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/files/(?P<file_id>[^/.]+)/download',
    viewset=StorageFileDownloadViewSet,
    basename=RouteNames.ROUTE_STORAGE_FILE_DOWNLOAD
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/trash',
    viewset=StorageTrashViewSet,
    basename=RouteNames.ROUTE_STORAGE_TRASH
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/images',
    viewset=StorageFileImageViewSet,
    basename=RouteNames.ROUTE_STORAGE_IMAGE
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/images/(?P<image_id>[^/.]+)/download',
    viewset=StorageFileImageDownloadViewSet,
    basename=RouteNames.ROUTE_STORAGE_IMAGE_DOWNLOAD
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/upload-from-file',
    viewset=StorageFileUploadView,
    basename=RouteNames.ROUTE_STORAGE_UPLOAD_FILE
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/upload-from-url',
    viewset=StorageFileUploadUrlView,
    basename=RouteNames.ROUTE_STORAGE_UPLOAD_URL
)
router.register(
    prefix=r'(?P<storage_id>[^/.]+)/users',
    viewset=StorageUserViewSet,
    basename=RouteNames.ROUTE_STORAGE_USER
)
