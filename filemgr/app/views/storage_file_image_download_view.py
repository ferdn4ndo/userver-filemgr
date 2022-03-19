from django.utils.datetime_safe import datetime
from rest_framework import viewsets, status
from rest_framework.generics import get_object_or_404
from rest_framework.request import Request
from rest_framework.response import Response

from app.models import StorageFileDownload, DownloadHit, StorageFileImage, StorageFileImageDimensions, Storage, StorageUser
from app.serializers import StorageFileDownloadSerializer
from app.services import policy
from app.services.translation import Messages


class StorageFileImageDownloadViewSet(viewsets.ViewSet):
    permission_classes = [policy.IsLoggedIn]

    def list(self, request: Request, storage_id: str, image_id: str, size_tag: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        tag = get_object_or_404(StorageFileImageDimensions.objects.all, tag=size_tag)
        image_file = get_object_or_404(StorageFileImage.create_storage_queryset(storage=storage, tag=tag), id=image_id)
        if not image_file.storage_file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        download_queryset = StorageFileDownload.objects.filter(owner=request.user, storage_file=image_file.storage_file)
        download_serializer = StorageFileDownloadSerializer(download_queryset, many=True)
        return Response(download_serializer.data)

    def create(self, request: Request, storage_id: str, image_id: str, size_tag: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        tag = get_object_or_404(StorageFileImageDimensions.objects.all, tag=size_tag)
        image_file = get_object_or_404(StorageFileImage.create_storage_queryset(storage=storage, tag=tag), id=image_id)
        if not image_file.storage_file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        serializer = StorageFileDownloadSerializer(data={'storage_file': image_file.storage_file.id, 'owner': request.user.id})
        serializer.is_valid(raise_exception=True)
        serializer.save()
        return Response(serializer.data, status=status.HTTP_201_CREATED)

    def retrieve(self, request: Request, storage_id: str, image_id: str, size_tag: str, pk: str) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        tag = get_object_or_404(StorageFileImageDimensions.objects.all, tag=size_tag)
        image_file = get_object_or_404(StorageFileImage.create_storage_queryset(storage=storage, tag=tag), id=image_id)
        if not image_file.storage_file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        download = get_object_or_404(StorageFileDownload.objects.filter(storage_file=image_file.storage_file), id=pk)
        if download.expires_at > datetime.now():
            return Response({"message": Messages.MSG_DOWNLOAD_EXPIRED}, status.HTTP_410_GONE)

        hit = DownloadHit.create_from_download_request(download=download, request=request)
        serializer = StorageFileDownloadSerializer(download)
        return Response(serializer.data, status.HTTP_200_OK)
