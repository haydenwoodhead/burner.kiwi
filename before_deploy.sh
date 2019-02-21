#!/bin/bash
#This script sets up the buildres directory into sub folders ready to be deployed to different buckets and functions

mkdir ./buildres/cloudformation
mkdir ./buildres/static

cp ./buildres/burnerkiwi ./buildres/cloudformation
cd ./buildres/cloudformation
zip burnerkiwi.zip burnerkiwi
cd ../
cd ../
cp cloudformation.json ./buildres/cloudformation/

mv ./buildres/*min.css ./buildres/static
mv ./buildres/*.svg ./buildres/static


