#!/usr/bin/env bash

uuid="$(uuid -v4)"
pandoc - -o "${uuid}.pdf" <<Here
Upload-Test

Kein Beleg

${uuid}
Here
