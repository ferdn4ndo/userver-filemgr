from rest_framework import viewsets, status
from rest_framework.generics import get_object_or_404
from rest_framework.request import Request
from rest_framework.response import Response

from app.models import StorageFileImage, Storage, StorageUser
from app.serializers import StorageFileImageSerializer
from app.services import policy
from app.services.translation import Messages


class StorageFileImageViewSet(viewsets.ViewSet):
    permission_classes = [policy.IsLoggedIn]

    def list(self, request: Request, storage_id: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        image_queryset = StorageFileImage.create_storage_queryset(storage=storage, user=request.user)
        serializer = StorageFileImageSerializer(image_queryset, many=True)
        return Response(serializer.data)

    def retrieve(self, request: Request, storage_id: str, pk: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        image_queryset = StorageFileImage.create_storage_queryset(storage=storage, user=request.user)
        image_file = get_object_or_404(image_queryset, id=pk)
        if not image_file.storage_file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        serializer = StorageFileImageSerializer(image_file)
        return Response(serializer.data, status.HTTP_200_OK)
