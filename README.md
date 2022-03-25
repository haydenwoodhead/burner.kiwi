![Roger the pyro kiwi](roger.png?raw=true "Meet Roger. They pyromaniac Kiwi.")

# burner.kiwi

![Build Status](https://github.com/haydenwoodhead/burner.kiwi/actions/workflows/qa.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/haydenwoodhead/burner.kiwi)](https://goreportcard.com/report/github.com/haydenwoodhead/burner.kiwi) [![Coverage Status](https://coveralls.io/repos/github/haydenwoodhead/burner.kiwi/badge.svg)](https://coveralls.io/github/haydenwoodhead/burner.kiwi)

A temporary email service and API built in Go. No JavaScript. No tracking. No analytics. No bullshit.

Check it out here: https://burner.kiwi

## About

Burner.kiwi is designed to be an easy to use, fast, and lightweight temporary mail service. It purposefully doesn't include tracking code, analytics, or advertising and has a beautiful and responsive UI.

For those wanting to self-host, burner.kiwi is designed to be able to run on both AWS Lambda and normal machines. It has several backing database implementations and can be flexibly configured.

There are three production-ready database implementations: DynamoDB, PostgreSQL and SQLite3.

There are also two email implementations: Mailgun and SMTP. SMTP allows you to receive emails directly at no extra cost but will not work with AWS lambda.

This is project still a work in progress, if you think you can help, see the To Do section.

## Deploy Your Own!

To run burner.kiwi yourself, you can build a binary or use the [official docker image](https://github.com/haydenwoodhead/burner.kiwi/pkgs/container/burner.kiwi).

## Build

To perform a build you must have [minify](https://github.com/tdewolff/minify/tree/master/cmd/minify) installed, and a working go installation.

```bash
make build
```

Or if you wish to use SQLite3 you must run (this enables CGO for sqlite, the binary produced is capable of connecting to any database):

```bash
make build-sqlite
```

## Test

Run all of burner.kiwi's tests locally:

```bash
make test
```

## Configuration Parameters

These are all set as environment variables.

### General

| Parameter     | Type     | Description                                                                                                                                                                  |
| ------------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| LISTEN        | string   | address to listen on. Default is `:8080` which is all interfaces, port 8080                                                                                                  |
| LAMBDA        | Boolean  | Whether or not the binary is being hosted on AWS Lambda                                                                                                                      |
| KEY           | String   | Secret key used to sign cookies and keys. Make this something strong!                                                                                                        |
| WEBSITE_URL   | String   | The url where the binary is being hosted. This must be internet reachable as it is the destination for Mailgun routes                                                        |
| STATIC_URL    | String   | The url where static content is being hosted. Set to `/static` to have the binary serve it. Otherwise set to a full domain name with protocol e.g https://static.example.com |
| DEVELOPING    | Boolean  | Set to `true` to disable HSTS and set `Cache-Control` to zero.                                                                                                               |
| DOMAINS       | []String | Comma separated list of domains connected to Mailgun account or that have correctly set MX records                                                                           |
| RESTOREREALIP | Boolean  | Restores the real remote ip using the `CF-Connecting-IP` header. Set to `true` to enable, `false` by default                                                                 |
| BLACKLISTED   | []String | Comma separated list of domains to reject email from                                                                                                                         |

### Email

| Parameter   | Type   | Description                                                          |
| ----------- | ------ | -------------------------------------------------------------------- |
| EMAIL_TYPE  | String | One of `mailgun` or `smtp`                                           |
| SMTP_LISTEN | String | Listen address for SMTP server (default 25)                          |
| MG_KEY      | String | Mailgun private API key (if using mailgun)                           |
| MG_DOMAIN   | String | One of the domains set up on your Mailgun account (if using mailgun) |

### Database

| Parameter    | Type   | Description                                                                                                                                                      |
| ------------ | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| DB_TYPE      | String | One of `memory`, `postgres`, `sqlite3` or `dynamo` for InMemory, PostgreSQL, SQLite3 (not this requires building with SQLite3 support) and DynamoDB respectively |
| DATABASE_URL | String | URL for the PostgreSQL database or filename for SQLite3 see [documentation here](https://github.com/mattn/go-sqlite3#dsn-examples).                              |
| DYNAMO_TABLE | String | Name of the dynamodb table to use for storage (if using DynamoDB)                                                                                                |

## AWS

If you are using DynamoDB in a non-AWS environment you need to set these. If you are on AWS you shouldg use IAM roles.

| Parameter             | Type   | Description                                                                                                                                                                 |
| --------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| AWS_ACCESS_KEY_ID     | String | Your AWS access key ID corresponding to an IAM role with permission to use DynamoDB                                                                                         |
| AWS_SECRET_ACCESS_KEY | String | AWS secret access key corresponding to your access key ID                                                                                                                   |
| AWS_REGION            | String | The AWS region containing the DynamoDB table. Use the appropriate value from the Region column [here](https://docs.aws.amazon.com/general/latest/gr/rande.html#ddb_region). |

## Contributing

If you notice any issues or have anything to add, I would be more than happy to work with you.
Create an issue and outline your plans or bugs.

## To do

- More tests for server package
- Proxy images to disrupt tracking pixels
- Attachments

If you think you can help, then create an issue and outline your plans.

## Contributors

Thanks to:

- cdubz for adding SQLite3 and custom address support
- lopezator for switching to go modules

## License

Copyright 2018-2022 Hayden Woodhead

Licensed under the MIT License.

The Roger logo is licensed under [Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International (CC BY-NC-SA 4.0)](https://creativecommons.org/licenses/by-nc-sa/4.0/).
