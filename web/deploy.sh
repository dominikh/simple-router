#!/bin/sh
go build web.go
scss screen.scss css/screen.css
~/.npm/coffee-script/1.3.3/package/bin/coffee -c internet.coffee
mv internet.js js/
scp web *.html admin@router:web/; scp -r js/ img/ css/ admin@router:/var/www/router/
