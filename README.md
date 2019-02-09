![Roger the pyro kiwi](roger.png?raw=true "Meet Roger. They pyromaniac Kiwi.")

# Burner.kiwi
[![Build Status](https://travis-ci.org/haydenwoodhead/burner.kiwi.svg?branch=master)](https://travis-ci.org/haydenwoodhead/burner.kiwi) [![Go Report Card](https://goreportcard.com/badge/github.com/haydenwoodhead/burner.kiwi)](https://goreportcard.com/report/github.com/haydenwoodhead/burner.kiwi) [![Coverage Status](https://coveralls.io/repos/github/haydenwoodhead/burner.kiwi/badge.svg)](https://coveralls.io/github/haydenwoodhead/burner.kiwi)

A temporary email service and API built in Go. No JavaScript. No tracking. No analytics. No bullshit.

Check it out here: https://burner.kiwi

## About

Burner.kiwi is designed to be able to run on both AWS Lambda and normal machines. The __goal__ is to have several backing 
database implementations and flexible configuration.

At this point it's working on normal machines and in Lambda. There are now two production-ready database implementation - DynamoDB & PostgreSQL
and a dev/testing implementation - InMemory.

This is definitely still a work in progress, see the To Do section.

## Deploy Your Own!

You will need to:
1. Buy at least one domain
2. Sign up for a Mailgun account
3. Add your new domain(s) to your Mailgun account
4. Ensure you can receive email through Mailgun on that domain

### AWS Lambda

Deploy your own straight to AWS Lambda and DynamoDB. 

Deploy to ap-southeast-2 (Sydney):

[![Deploy](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/new?stackName=burnerkiwi&templateURL=https://s3-ap-southeast-2.amazonaws.com/burner-kiwi-ap-southeast-2/cloudformation.json)

Deploy to us-east-1 (N. Virginia):

[![Deploy](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/new?stackName=burnerkiwi&templateURL=https://s3.amazonaws.com/burner-kiwi-us-east-1/cloudformation.json)

Deploy to eu-west-1 (Ireland):

[![Deploy](https://s3.amazonaws.com/cloudformation-examples/cloudformation-launch-stack.png)](https://console.aws.amazon.com/cloudformation/home?region=eu-west-1#/stacks/new?stackName=burnerkiwi&templateURL=https://s3-eu-west-1.amazonaws.com/burner-kiwi-eu-west-1/cloudformation.json)

If you want to deploy to another AWS region you will modify the provided cloudformation.json template and upload your code to a bucket in that region.

### Other

Or run it on your own server. Build a binary, set up with the configuration parameters detailed below and run it.

## Build

### Development

To build for development just run `go build` - nothing special here.

### Production

Building for production is a different beast. We need to give assets cache friendly names, minify them and add their 
new names to the binary. Check out the included `build.sh` file.

To build run `./build.sh` from the root of the project. This will create and populate a `buildres` directory
containing the binary file and minified/renamed static assets.

## Configuration Parameters

These are all set as environment variables.

Parameter | Type | Description
----------|------|-------------
LAMBDA | Boolean | Whether or not the binary is being hosted on AWS Lambda
KEY | String | Key used to sign cookies and keys. Make this something strong!
WEBSITE_URL | String | The url where the binary is being hosted. This must be internet reachable as it is the destination for Mailgun routes
STATIC_URL | String | The url where static content is being hosted. Set to `/static` to have the binary serve it. Otherwise set to a full domain name with protocol e.g https://static.example.com
DEVELOPING | Boolean | Set to `true` to disable HSTS and set `Cache-Control` to zero. 
DOMAINS | []String | Comma separated list of domains connected to Mailgun account and able to receive email
MG_KEY | String | Mailgun private API key
MG_DOMAIN | String | One of the domains set up on your Mailgun account
DB_TYPE | String | One of `memory`, `postgres` or `dynamo` for InMemory, PostgreSQL and DynamoDB respectively 
DATABASE_URL | String | URL for the PostgreSQL database 
DYNAMO_TABLE | String | Name of the dynamodb table to use for storage (if using DynamoDB)
RESTOREREALIP | String | Restores the real remote ip using the `CF-Connecting-IP header`. Set to `true` to enable

If you are using DynamoDB a non AWS environment you need to set these. If you are on AWS you should, of course, should use IAM roles.

Parameter | Type | Description
----------|------|-------------
AWS_ACCESS_KEY_ID | String | Your AWS access key ID corresponding to an IAM role with permission to use DynamoDB
AWS_SECRET_ACCESS_KEY | String | AWS secret access key corresponding to your access key ID
AWS_REGION | String | The AWS region containing the DynamoDB table. Use the appropriate value from the Region column [here](https://docs.aws.amazon.com/general/latest/gr/rande.html#ddb_region).

## Deleting Old Routes

Burner.kiwi creates a new Mailgun route for every inbox and email address. This allows us to delete these routes once the
inbox expires and prevents the server being unnecessarily burdened by webhooks for inboxes that don't exist anymore. 

Old routes are deleted in the background every time the binary is started. In a Lambda context this means every time we 
have a cold start, the CloudFormation template sets up a CloudWatch event to call the handler every 6 hours. However, because
of the fact that burner.kiwi is designed to be platform agnostic we can't differentiate between these CloudWatch events and 
normal http requests. This means if the CloudWatch event hits a frozen container rather than causing a new container to
be spawned, we wont trigger the deletion of old routes. Hopefully, a normal http request or CloudWatch event will cause 
a new container to be spawned often enough that old routes are cleared out. 

If you are deployed on a normal machine you can explicitly cause deletion of old routes without starting the http server. 
You can set up a cron to run the binary every 6 hours, like so: `burnerkiwi -delete-old-routes`. You will need to ensure
the binary can still access the environment variables. 

## Contributing

If you notice any issues or have anything to add, I would be more than happy to work with you. 
Create an issue and outline your plans or bugs.

## To do

* Code refactor/redo 
* More tests for server package
* More database implementations (PSQL, SQLite, etc)
* Night theme
* Print html errors rather than just plain text
* Better configuration
* Noob friendly setup tutorial

Again, if you think you can help, then create an issue and outline your plans.

## License

Copyright 2018 Hayden Woodhead

Licensed under the MIT License. 

The Roger logo is drawn by Melissa Bather, used with permission, and licensed under 
[Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0)](https://creativecommons.org/licenses/by-nc-sa/4.0/).