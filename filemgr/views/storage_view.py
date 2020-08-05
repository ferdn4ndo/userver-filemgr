from filemgr.models import Storage
from filemgr.serializers import StorageSerializer
from filemgr.services import policy
from .generic_model_view import FullCRUDListModelViewSet


class StorageViewSet(FullCRUDListModelViewSet):
    queryset = Storage.objects.all()
    serializer_class = StorageSerializer
    permission_classes = [policy.IsLoggedIn, policy.IsAdminOrDeny]
