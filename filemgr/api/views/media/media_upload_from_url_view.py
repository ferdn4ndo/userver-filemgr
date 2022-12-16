from django.http import JsonResponse

from core.services.web_request.web_request_service import WebRequestService


@api_view(["POST"])
def create_from_url(request):
    if 'url' not in request.body:
        return JsonResponse({'message': "Missing the 'url' key in body"}, status=400)

    url = request.body['url']

    service = WebRequestService(url=url)

    if not service.is_downloadable():
        return JsonResponse({'message': "The given URL "}, status=400)

    file_uuid = get_new_unique_file_identifier('media')
    uploaded_url = upload_from_url(url, file_uuid, 'media')

    return JsonResponse({
        'message': 'OK',
        'fileUrl': uploaded_url,
    })
