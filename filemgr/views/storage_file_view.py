from django.db.models import QuerySet
from rest_framework import status
from rest_framework.request import Request
from rest_framework.response import Response

from filemgr.models import StorageFile, Storage, StorageUser, get_object_or_404
from filemgr.serializers import StorageFileSerializer
from filemgr.services import policy
from filemgr.services.translation import Messages
from filemgr.views.generic_model_view import ReadUpdateDestroyModelViewSet


class StorageFileViewSet(ReadUpdateDestroyModelViewSet):
    permission_classes = [policy.IsLoggedIn, policy.IsAdminOrOwnerOrReadOnly]
    serializer_class = StorageFileSerializer

    def get_queryset(self) -> QuerySet:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        queryset = StorageFile.objects.filter(storage=storage)
        return queryset

    def list(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)
        return super(StorageFileViewSet, self).list(request=request, *args, **kwargs)

    def retrieve(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)
        return super(StorageFileViewSet, self).retrieve(request=request, *args, **kwargs)

    def partial_update(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)
        return super(StorageFileViewSet, self).partial_update(request=request, *args, **kwargs)

    def destroy(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), pk=self.kwargs['storage_id'])
        if not StorageUser.userMayWriteStorage(self.request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_WRITE_PERM}, status=status.HTTP_403_FORBIDDEN)

        file = get_object_or_404(self.get_queryset(), id=self.kwargs['pk'])
        file.excluded = True
        file.updated_by = request.user
        file.save()
        return Response(status=status.HTTP_204_NO_CONTENT)
