from django.http import HttpResponse
from django.http import JsonResponse
from django.utils.decorators import method_decorator
from django.views.decorators.csrf import csrf_exempt
from rest_framework import authentication
from rest_framework.permissions import IsAuthenticated
from rest_framework.views import APIView


def index(request):
    return HttpResponse("Hello, world. You're at the views index.")


@method_decorator(csrf_exempt, name='dispatch')
class UploadMedia(APIView):
    permission_classes = [IsAuthenticated]
    authentication_classes = [authentication.TokenAuthentication]

    def post(self, request):
        if 'file' not in request.FILES:
            return JsonResponse({'message': 'Missing file field in form data'}, status=400)

        # ToDo: fix this
        print(request.POST)
        return JsonResponse([])
        #return JsonResponse(MediaItemSerializer(media_item).data)
