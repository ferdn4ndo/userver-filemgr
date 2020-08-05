from rest_framework import viewsets, status
from rest_framework.generics import get_object_or_404
from rest_framework.request import Request
from rest_framework.response import Response

from filemgr.models import StorageFile, Storage, StorageUser
from filemgr.serializers import StorageFileSerializer
from filemgr.services import policy
from filemgr.services.translation import Messages


class StorageTrashViewSet(viewsets.ViewSet):
    permission_classes = [policy.IsLoggedIn]

    def list(self, request: Request, storage_id: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)
        queryset = StorageFile.create_storage_queryset(user=request.user, storage=storage, excluded=True)
        serializer = StorageFileSerializer(queryset, many=True)
        return Response(serializer.data)

    def retrieve(self, request: Request, storage_id: str, pk: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)
        queryset = StorageFile.create_storage_queryset(user=request.user, storage=storage, excluded=True)
        file = get_object_or_404(queryset, pk=pk)
        serializer = StorageFileSerializer(file)
        return Response(serializer.data, status.HTTP_200_OK)

    def destroy(self, request: Request, storage_id: str, pk: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)
        queryset = StorageFile.create_storage_queryset(user=request.user, storage=storage, excluded=True)
        file = get_object_or_404(queryset, pk=pk)
        file.delete()
        return Response(status=status.HTTP_204_NO_CONTENT)
