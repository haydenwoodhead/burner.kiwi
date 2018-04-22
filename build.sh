#!/bin/bash
# This script builds the burner.kiwi project producing a binary file and output css files

# This script requires that github.com/tdewolff/minify/tree/master/cmd/minify is installed
if ! type minify > /dev/null; then
    echo Minify not installed. Please go to: github.com/tdewolff/minify/tree/master/cmd/minify
    exit 1
fi

# This script requires that https://github.com/gobuffalo/packr is installed
if ! type packr > /dev/null; then
    echo Packr not installed. Please go to: https://github.com/gobuffalo/packr
    exit 1
fi

# Create build res folder
mkdir ./buildres

# Get hash of minified files
custom_hash=`md5sum ./static/custom.css | cut -c -32`
milligram_hash=`md5sum ./static/milligram.css | cut -c -32`
normalize_hash=`md5sum ./static/normalize.css | cut -c -32`

custom_name="custom.$custom_hash.min.css"
milligram_name="milligram.$milligram_hash.min.css"
normalize_name="normalize.$normalize_hash.min.css"

# Minify CSS - yes I'm aware that I'm getting the hash before minifying
# but this renaming is for the purposes of cache busting not verifying files are correct
minify -o "./static/$custom_name" ./static/custom.css
minify -o "./static/$milligram_name" ./static/milligram.css
minify -o "./static/$normalize_name" ./static/normalize.css

git_commit=`git rev-parse --short HEAD`

# Build the go binary with build flags to override vars
packr build -ldflags "-X github.com/haydenwoodhead/burnerkiwi/server.version=${git_commit} -X github.com/haydenwoodhead/burnerkiwi/server.milligram=${milligram_name} -X github.com/haydenwoodhead/burnerkiwi/server.custom=${custom_name} -X github.com/haydenwoodhead/burnerkiwi/server.normalize=${normalize_name}" -o "./buildres/burnerkiwi-$git_commit"

# Move css files to build result
mv "./static/$custom_name" ./buildres/
mv "./static/$milligram_name" ./buildres/
mv "./static/$normalize_name" ./buildres/
cp ./static/logo-placeholder.png ./buildres/

echo Build Complete
exit 0