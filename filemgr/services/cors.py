class CorsMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response
        # One-time configuration and initialization.

    def __call__(self, request):
        # Code to be executed for each request before
        # the view (and later middleware) are called.

        response = self.get_response(request)

        # Code to be executed for each request/response after the view is called.
        response["Access-Control-Allow-Origin"] = request.headers['Origin'] if 'Origin' in request.headers else '*'

        # In addition to the CORS-safelisted request headers, the following are also allowed
        allowed_headers = [
            "Content-Type",
            "Authorization",
        ]
        response["Access-Control-Allow-Headers"] = ", ".join(allowed_headers)

        response["Access-Control-Allow-Methods"] = "*"
        response["Access-Control-Allow-Credentials"] = "true"
        response["Access-Control-Max-Age"] = 7200

        return response
