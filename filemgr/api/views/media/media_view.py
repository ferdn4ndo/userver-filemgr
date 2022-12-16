from rest_framework import generics, authentication

from api.serializers.media.media_serializer import MediaSerializer
from core.models import Media


class MediaView(generics.ListAPIView):
    queryset = Media.objects.all()
    serializer_class = MediaSerializer
    authentication_classes = [authentication.TokenAuthentication]
