from django.http import Http404
from rest_framework import exceptions, status
from rest_framework.response import Response
from rest_framework.views import set_rollback

from filemgr.errors import NotFoundError
from filemgr.services.translation import Messages


class NotFoundException(exceptions.APIException):
    status_code = status.HTTP_404_NOT_FOUND
    default_detail = Messages.MSG_NOT_FOUND
    default_code = 'not_found'


class PermissionDeniedException(exceptions.APIException):
    status_code = status.HTTP_403_FORBIDDEN
    default_detail = Messages.MSG_NOT_ENOUGH_PERMS
    default_code = 'permission_denied'


class NotAuthenticatedException(exceptions.APIException):
    status_code = status.HTTP_401_UNAUTHORIZED
    default_detail = Messages.MSG_NOT_AUTHENTICATED
    default_code = 'not_authenticated'


class BadRequestException(exceptions.APIException):
    status_code = status.HTTP_400_BAD_REQUEST
    default_detail = Messages.MSG_INVALID_INPUT_DATA
    default_code = 'invalid'

    def __init__(self, detail=None):
        if detail is None:
            detail = self.default_detail

        # For validation failures, we may collect many errors together,
        # so the details should always be coerced to a list if not already.
        if not isinstance(detail, dict) and not isinstance(detail, list):
            detail = [detail]

        if not isinstance(detail, dict):
            self.detail = detail
            return

        self.detail = [
            {
                "field": validation_error_field,
                "error": detail[validation_error_field]
            } for validation_error_field in detail.keys()
        ]


def custom_exception_handler(exc: Exception, context):
    # The context argument is not used by the default handler, but can be useful if the exception handler needs
    # further information such as the view currently being handled, which can be accessed as context['view'].

    if isinstance(exc, NotFoundError) or isinstance(exc, Http404):
        exc = NotFoundException()
    elif isinstance(exc, exceptions.PermissionDenied):
        exc = PermissionDeniedException()
    elif isinstance(exc, exceptions.NotAuthenticated):
        exc = NotAuthenticatedException()
    elif isinstance(exc, exceptions.ValidationError):
        exc = BadRequestException(detail=exc.detail)

    if isinstance(exc, exceptions.APIException):
        headers = {}
        if hasattr(exc, 'auth_header'):
            headers['WWW-Authenticate'] = getattr(exc, 'auth_header')
        if hasattr(exc, 'wait'):
            headers['Retry-After'] = '%d' % getattr(exc, 'wait')

        if isinstance(exc.detail, (list, dict)):
            data = {
                'message': Messages.MSG_ONE_OR_MORE_ERRORS_OCCURRED,
                'errors': exc.detail
            }
        else:
            data = {'message': exc.detail}

        set_rollback()
        return Response(data, status=exc.status_code, headers=headers)

    return None
