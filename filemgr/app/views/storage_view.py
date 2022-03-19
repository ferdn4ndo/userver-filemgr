from app.models import Storage
from app.serializers import StorageSerializer
from app.services import policy

from .generic_model_view import FullCRUDListModelViewSet


class StorageViewSet(FullCRUDListModelViewSet):
    queryset = Storage.objects.all()
    serializer_class = StorageSerializer
    permission_classes = [policy.IsLoggedIn, policy.IsAdminOrDeny]
