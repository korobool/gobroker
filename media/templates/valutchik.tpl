<html>
  <head>
    {% if meta %}
        {% for key, value in meta.items() %}
            <meta name="{{ key }}" content="{{ value }}">
        {% endfor %}
    {% endif %}

    <meta name="viewport" content="width=device-width, initial-scale=1.0">

    <link rel="stylesheet" href="/css/smartbanner.css" type="text/css" media="screen">
    
    <script src="/js/jquery-2.1.4.min.js"></script>
    <script src="/js/smartbanner.v1.js"></script>
    <script>$(function () { $.smartbanner({}); });</script>
  </head>
  <body>
    <img class="landing" src="/img/{{ image }}">
  </body>
</html>