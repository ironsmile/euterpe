#!/bin/bash

# This script will watch for changes in the .css or .js files of the project
# and rebuild the "squashed" versions which are to be released.

find . -type f -name '*.css' -or -name '*.js' | grep -v '.min.' | entr ./http_root/squash
