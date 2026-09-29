package main

const help = `
Console HTTP client for interfacing with epox server.

Common flags:

  -u (string) -  url of server api
  -a (string) -  auth token
  -k (string) -  crypto key  (optional)
  -f (string) -  config file (optional)

Default config file is epox.conf, if it is not present then
flags -u and -a must be set for each command.

Flag combinations:

  -t <topic>
    Display messages from topic with latest offset.

  -t <topic> -o <offset>
    Display messages from topic with specified offset.

  -t <topic> -m <file>
    Add message to topic. Text is taken from file.

  -t <topic> -c
    Create new topic.

  -t <topic> -i
    Display topic info.

When -t flag is not set client requests list of all topics.
`
