# Delete Helper

This directory contains a helper lambda function. The function is triggered by a cloudwatch event every 6 hours and 
calls the endpoint `/bk/deleteoldroutes/` with the appropriate headers.

This exists because I haven't figured out how to distinguish between a cloudwatch event and a apigateway request in
the current main.go and server.go code (I dont think this is possible when using apex/gateway). This means I can't setup the cloudwatch event to point
to the function running burner.kiwi. So the easiest solution is to setup a new endpoint and call it
from a different lambda function.  