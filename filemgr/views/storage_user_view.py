from django.db.models import QuerySet
from rest_framework.request import Request
from rest_framework.response import Response

from filemgr.models import Storage, StorageUser, get_object_or_404
from filemgr.serializers import StorageUserSerializer
from filemgr.services import policy
from .generic_model_view import FullCRUDListModelViewSet


class StorageUserViewSet(FullCRUDListModelViewSet):
    permission_classes = [policy.IsLoggedIn, policy.IsAdminOrDeny]
    serializer_class = StorageUserSerializer

    def get_queryset(self) -> QuerySet:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        queryset = StorageUser.objects.filter(storage=storage)
        return queryset

    def create(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        request.data['storage'] = str(storage.id)
        request.data['created_by'] = request.user.username
        return super(StorageUserViewSet, self).create(request=request, *args, **kwargs)

    def partial_update(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        request.data['storage'] = str(storage.id)
        request.data['updated_by'] = request.user.username
        return super(StorageUserViewSet, self).partial_update(request=request, *args, **kwargs)
