#!/bin/bash

heroku ps:scale web=0 -a fpe-go
heroku container:push web -a fpe-go
heroku container:release web -a fpe-go
heroku ps:scale web=1 -a fpe-go
heroku logs -a fpe-go