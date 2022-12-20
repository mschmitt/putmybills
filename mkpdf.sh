#!/usr/bin/env bash

randid="$(openssl rand -hex 2)"
printf -v pdffile "test-%s.pdf" "${randid}"
pandoc - -o "${pdffile}" <<Here
Upload-Test

Kein Beleg

${randid}
Here
