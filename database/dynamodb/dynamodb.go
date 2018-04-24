package dynamodb

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/haydenwoodhead/burnerkiwi/server"
)

// DynamoDB implements the db interface
type DynamoDB struct {
	dynDB                 *dynamodb.DynamoDB
	emailsTableName       string
	emailAddressIndexName string
	messagesTableName     string
}

//GetNewDynamoDB gets a new dynamodb database or panics
func GetNewDynamoDB() *DynamoDB {
	awsSession := session.Must(session.NewSession())
	dynDB := dynamodb.New(awsSession)

	return &DynamoDB{
		dynDB:                 dynDB,
		emailsTableName:       "bk-emails",
		emailAddressIndexName: "email_address-index",
		messagesTableName:     "bk-messages",
	}
}

// SaveNewInbox saves a given inbox to dynamodb
func (d *DynamoDB) SaveNewInbox(i server.Inbox) error {
	av, err := dynamodbattribute.MarshalMap(i)

	if err != nil {
		return fmt.Errorf("SaveNewInbox: failed to marshal struct to attribute value: %v", err)

	}

	_, err = d.dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.emailsTableName),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("SaveNewInbox: failed to put to dynamodb: %v", err)
	}

	return nil
}

//GetInboxByID gets an inbox by the given inbox id
func (d *DynamoDB) GetInboxByID(id string) (server.Inbox, error) {
	var i server.Inbox

	o, err := d.dynDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		TableName: aws.String(d.emailsTableName),
	})

	if err != nil {
		return server.Inbox{}, err
	}

	err = dynamodbattribute.UnmarshalMap(o.Item, &i)

	if err != nil {
		return server.Inbox{}, err
	}

	return i, nil
}

//EmailAddressExists returns a bool depending on whether or not the given email address
// is already assigned to an inbox
func (d *DynamoDB) EmailAddressExists(a string) (bool, error) {
	q := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("email_address = :e"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":e": {
				S: aws.String(a),
			},
		},
		IndexName: aws.String(d.emailAddressIndexName),
		TableName: aws.String(d.emailsTableName),
	}

	res, err := d.dynDB.Query(q)

	if err != nil {
		return false, err
	}

	if len(res.Items) == 0 {
		return false, nil
	}

	return true, nil
}

// SetInboxCreated updates the given inbox to reflect its created status
func (d *DynamoDB) SetInboxCreated(i server.Inbox) error {
	u := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#F": aws.String("failed_to_create"),
			"#M": aws.String("mg_routeid"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":f": {
				BOOL: aws.Bool(false),
			},
			":m": {
				S: aws.String(i.MGRouteID),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(i.ID),
			},
		},
		TableName:        aws.String(d.emailsTableName),
		UpdateExpression: aws.String("SET #F = :f, #M = :m"),
	}

	_, err := d.dynDB.UpdateItem(u)

	if err != nil {
		return fmt.Errorf("SetInboxCreated: failed to mark email as created: %v", err)
	}

	return nil
}

//SaveNewMessage saves a given message to dynamodb
func (d *DynamoDB) SaveNewMessage(m server.Message) error {
	mv, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		return fmt.Errorf("SaveMessage: failed to marshal struct to attribute value: %v", err)
	}

	_, err = d.dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(d.messagesTableName),
		Item:      mv,
	})

	if err != nil {
		return fmt.Errorf("SaveMessage: failed to put to dynamodb: %v", err)
	}

	return nil
}

//GetMessagesByInboxID returns all messages in a given inbox
func (d *DynamoDB) GetMessagesByInboxID(i string) ([]server.Message, error) {
	var m []server.Message

	qi := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(i),
			},
		},
		KeyConditionExpression: aws.String("inbox_id = :id"),
		TableName:              aws.String(d.messagesTableName),
	}

	res, err := d.dynDB.Query(qi)

	if err != nil {
		return []server.Message{}, fmt.Errorf("GetAllMessagesByInboxID: failed to query dynamodb: %v", err)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &m)

	if err != nil {
		return []server.Message{}, fmt.Errorf("GetAllMessagesByInboxID: failed to unmarshal result: %v", err)
	}

	return m, nil
}

//GetMessageByID gets a single message by the given inbox and message id
func (d *DynamoDB) GetMessageByID(i, m string) (server.Message, error) {
	var msg server.Message

	o, err := d.dynDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"inbox_id": {
				S: aws.String(i),
			},
			"message_id": {
				S: aws.String(m),
			},
		},
		TableName: aws.String(d.messagesTableName),
	})

	if err != nil {
		return server.Message{}, fmt.Errorf("GetMessageByID: failed to get message: %v", err)
	}

	err = dynamodbattribute.UnmarshalMap(o.Item, &msg)

	if err != nil {
		return server.Message{}, fmt.Errorf("GetMessageByID: failed to unmarshal message: %v", err)
	}

	if strings.Compare(msg.ID, "") == 0 {
		return server.Message{}, server.ErrMessageDoesntExist
	}

	return msg, nil
}

//createDatabase creates a new database for testing
func (d *DynamoDB) createDatabase() error {
	emails := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("email_address"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
			},
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: aws.String(d.emailAddressIndexName),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("email_address"),
						KeyType:       aws.String("HASH"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String(dynamodb.ProjectionTypeKeysOnly),
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(5),
					WriteCapacityUnits: aws.Int64(5),
				},
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String(d.emailsTableName),
	}

	_, err := d.dynDB.CreateTable(emails)

	if err != nil {
		return err
	}

	messages := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("inbox_id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("message_id"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("inbox_id"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("message_id"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
		TableName: aws.String(d.messagesTableName),
	}

	_, err = d.dynDB.CreateTable(messages)

	if err != nil {
		return err
	}

	return nil
}
