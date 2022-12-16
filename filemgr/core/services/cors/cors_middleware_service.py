class CorsMiddlewareService():
    def __init__(self, get_response):
        self.get_response = get_response
        # One-time configuration and initialization.

    def __call__(self, request):
        # Code to be executed for each request before
        # the view (and later middleware) are called.

        response = self.get_response(request)

        # Code to be executed for each request/response after
        # the view is called.

        # Prepares the list of allowed hosts to determine the origin response
        allowed_hosts = [
            'http://localhost:5000',
            'http://localhost:8080',
            'http://infotrem.lan',
            'http://api.infotrem.lan',
            'http://app.infotrem.lan',
            'http://api.infotrem.com.br',
            'http://app.infotrem.com.br',
            'http://infotrem.com.br',
            'https://api.infotrem.com.br',
            'https://app.infotrem.com.br',
            'https://infotrem.com.br',
            'https://infotrem.fconstantino.com.br',
            'https://app.infotrem.fconstantino.com.br',
            'https://api.infotrem.fconstantino.com.br',
        ]
        origin = request.headers['Origin'] if 'Origin' in request.headers else ''
        response['Access-Control-Allow-Origin'] = origin if origin in allowed_hosts else 'null'

        # In addition to the CORS-safelisted request headers, the following are also allowed
        allowed_headers = [
            'Content-Type',
            'Authorization',
        ]
        response['Access-Control-Allow-Headers'] = ', '.join(allowed_headers)

        response['Access-Control-Allow-Methods'] = '*'
        response['Access-Control-Allow-Credentials'] = 'true'
        response['Access-Control-Max-Age'] = 7200

        return response
