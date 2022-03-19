from django.utils.decorators import method_decorator
from django.views.decorators.csrf import csrf_exempt
from rest_framework import status, viewsets
from rest_framework.generics import get_object_or_404
from rest_framework.request import Request
from rest_framework.response import Response

from app.drivers import load_storage_driver
from app.models import Storage, StorageUser
from app.serializers import StorageFileSerializer, StorageFileUploadSerializer, StorageFileUploadUrlSerializer
from app.services import policy
from app.services.translation import Messages


@method_decorator(csrf_exempt, name='dispatch')
class StorageFileUploadView(viewsets.ViewSet):
    permission_classes = [policy.IsLoggedIn]

    def create(self, request: Request, storage_id: str) -> Response:
        if 'file' not in request.FILES:
            return Response({'message': Messages.MSG_MISSING_FILE_FIELD_FORM}, status=status.HTTP_400_BAD_REQUEST)
        request_file = request.FILES['file']

        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayWriteStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_WRITE_PERM}, status=status.HTTP_403_FORBIDDEN)

        serializer = StorageFileUploadSerializer(data=request.POST)
        serializer.is_valid(raise_exception=True)

        driver = load_storage_driver(storage)
        data = serializer.validated_data

        try:
            file = driver.upload_from_request_file(
                user=request.user,
                request_file=request_file,
                visibility=data['visibility'],
                virtual_path=data['virtual_path'],
                overwrite=data['overwrite'],
            )
        except FileExistsError:
            return Response({'message': Messages.MGS_FILE_EXISTS_NO_OVERWRITE}, status=status.HTTP_409_CONFLICT)

        file_serializer = StorageFileSerializer(file)
        return Response(file_serializer.data, status.HTTP_201_CREATED)


class StorageFileUploadUrlView(viewsets.ViewSet):
    permission_classes = [policy.IsLoggedIn]

    def create(self, request: Request, storage_id: str) -> Response:
        serializer = StorageFileUploadUrlSerializer(data=request.data)
        serializer.is_valid(raise_exception=True)

        storage = get_object_or_404(Storage.objects.all(), id=storage_id)
        if not StorageUser.userMayWriteStorage(request.user, storage):
            return Response({'message': Messages.MSG_NO_STORAGE_WRITE_PERM}, status=status.HTTP_403_FORBIDDEN)

        driver = load_storage_driver(storage)
        data = serializer.validated_data
        file = driver.upload_from_path(
            user=request.user,
            path=data['url'],
            virtual_path=data['virtual_path'],
            overwrite=data['overwrite'],
            visibility=data['visibility'],
            is_url=True
        )

        file_serializer = StorageFileSerializer(file)
        return Response(file_serializer.data, status.HTTP_201_CREATED)
