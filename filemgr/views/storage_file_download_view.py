from django.db.models import QuerySet
from rest_framework import viewsets, status
from rest_framework.generics import get_object_or_404
from rest_framework.request import Request
from rest_framework.response import Response

from filemgr.models import StorageFileDownload, DownloadHit, StorageFile, Storage, StorageUser
from filemgr.serializers import StorageFileDownloadSerializer
from filemgr.services import policy
from filemgr.services.translation import Messages
from filemgr.views.generic_model_view import CreateReadModelViewSet


class StorageFileDownloadViewSet(CreateReadModelViewSet):
    permission_classes = [policy.IsLoggedIn]
    serializer_class = StorageFileDownloadSerializer

    def get_queryset(self):
        storage = get_object_or_404(Storage.objects.all(), id=self.kwargs['storage_id'])
        file = get_object_or_404(
            StorageFile.create_storage_queryset(user=self.request.user, storage=storage),
            id=self.kwargs['file_id']
        )
        queryset = StorageFileDownload.objects.filter(owner=self.request.user, storage_file=file)
        return queryset

    def list(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=kwargs['storage_id'])
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        file = get_object_or_404(
            StorageFile.create_storage_queryset(user=request.user, storage=storage),
            id=kwargs['file_id']
        )
        if not file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        return super(StorageFileDownloadViewSet, self).list(request=request, *args, **kwargs)

    def create(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=kwargs['storage_id'])
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        file = get_object_or_404(
            StorageFile.create_storage_queryset(user=request.user, storage=storage),
            id=kwargs['file_id']
        )
        if not file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        serializer = StorageFileDownloadSerializer(data={'storage_file': file.id, 'owner': request.user.id})
        serializer.is_valid(raise_exception=True)
        serializer.save()
        return Response(serializer.data, status=status.HTTP_201_CREATED)

    def retrieve(self, request: Request, *args, **kwargs) -> Response:
        storage = get_object_or_404(Storage.objects.all(), id=kwargs['storage_id'])
        if not StorageUser.userMayReadStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_READ_PERM}, status=status.HTTP_403_FORBIDDEN)

        file = get_object_or_404(
            StorageFile.create_storage_queryset(user=request.user, storage=storage),
            id=kwargs['file_id']
        )
        if not file.is_visible_by_user(request.user):
            return Response({'message': Messages.MSG_NO_FILE_DOWNLOAD_PERM}, status=status.HTTP_403_FORBIDDEN)

        download = get_object_or_404(StorageFileDownload.objects.filter(storage_file=file), id=kwargs['pk'])
        serializer = StorageFileDownloadSerializer(download)
        if serializer.data['expired']:
            return Response({"message": Messages.MSG_DOWNLOAD_EXPIRED}, status.HTTP_410_GONE)

        hit = DownloadHit.create_from_download_request(download=download, request=request)

        return Response(serializer.data, status.HTTP_200_OK)
