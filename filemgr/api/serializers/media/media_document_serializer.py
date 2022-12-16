from api.serializers.generic_audited_model_serializer import GenericAuditedModelSerializer
from core.models.media.media_document_model import MediaDocument


class MediaDocumentSerializer(GenericAuditedModelSerializer):

    class Meta:
        model = MediaDocument
        fields = [
            'id',
            'pages',
            'black_and_white',
            'created_at',
            'updated_at',
        ]
