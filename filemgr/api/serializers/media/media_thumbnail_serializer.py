from api.serializers.generic_audited_model_serializer import GenericAuditedModelSerializer
from core.models.media.media_thumbnail_model import MediaThumbnail


class MediaThumbnailSerializer(GenericAuditedModelSerializer):

    class Meta:
        model = MediaThumbnail
        fields = [
            'id',
            'media',
            'size_tag',
            'storage_file',
        ]
