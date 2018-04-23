# Burner.kiwi

A temporary email service and api built in golang. No javascript. No tracking. No analytics. No bullshit.

Check it out here: https://burner.kiwi

## About

Burner.kiwi is designed to be able to run on both AWS lambda and normal machines. The __goal__ is to have several backing 
database implementations and flexible configuration.

At this point it's working on normal machines and in lambda. There are two database implementations - DynamoDB and InMemory.
Configuration still needs some work. 

## Deploy Your Own!

Deploy your own straight to AWS lambda and DynamoDB. Use this button or the included cloudformation template.

Otherwise grab a binary from the releases page, setup with the configuration parameters detailed below and run it.

You will also need to:
1. Buy at least one domain
2. Sign up for a mailgun account
3. Add your new domain(s) to your mailgun account
4. Ensure you can receive email through mailgun on that domain

## Build

### Development

To build for development just run `go build` nothing special here.

### Production

Building for production is a different beast. We need to give assets cache friendly names and minify them and add build time
variables to the binary. Check out the included `build.sh` file.

To build run `./build.sh` from the root of the project. This will create and populate a `buildres` directory
containing the binary file and minified and rename static assets.

## Configuration Parameters

These are all set as environment variables.

Parameter | Type | Description
----------|------|-------------
LAMBDA | Boolean | Whether or not the binary is being hosted on AWS Lambda
KEY | String | Key used to sign cookies and keys. Make this something strong!
WEBSITE_URL | String | The url where the binary is being hosted. This must be internet reachable as it is the destination for mailgun routes
STATIC_URL | String | The url where static content is being hosted. Set to `/static` to have the binary serve it. Otherwise set to a full domain name with protocol e.g https://static.example.com
DEVELOPING | Boolean | Set to `true` to disable HSTS and set `Cache-Control` to zero. 
DOMAINS | []String | Comma separated list of domains connected to mailgun account and able to receive email
MG_KEY | String | Mailgun private api key
MG_DOMAIN | String | One of the domains setup on your mailgun account


If you are using DynamoDB a non AWS environment you need to set these. If you are on AWS you should of course should use IAM roles.

Parameter | Type | Description
----------|------|-------------
AWS_ACCESS_KEY_ID | String | Your AWS access key ID corresponding to an IAM role with permission to use DynamoDB
AWS_SECRET_ACCESS_KEY | String | AWS secret access key corresponding to your access key id
AWS_REGION | String | The AWS region containing the DynamoDB table. Use the appropriate value from the Region column [here](https://docs.aws.amazon.com/general/latest/gr/rande.html#ddb_region).

## Deleting Old Routes

If you do decide to deploy this yourself not using the cloudformation template you will need to setup some form of cron 
to tell burner.kiwi to delete old mailgun routes.

If you are deployed on lambda: if you used the cloudformation template you're sweet. Otherwise setup a cloudformation event
to call the lambda func every 6 hours or so. 

If you are deployed on a normal machine: setup a cron to run the binary every 6 hours or so like so `burnerkiwi -delete-old-routes`. You 
will need to ensure the binary can still access the environment variables. 

These will both trigger the removal of old routes from mailgun.

## Contributing

If you notice any issues or have anything to add I would be more than happy to work with you. 
Create an issue and outline your plans or bugs.

## Todo

* More database implementations (PSQL, SQLite, etc)
* Night theme
* Print html errors rather than just plain text
* More tests for http handlers
* Better configuration
* Noob friendly setup tutorial

Again if you think you can help create an issue and outline your plans.

## License

Copyright 2018 Hayden Woodhead

Licensed under the MIT License. 