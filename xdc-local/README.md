Installation:

Install Go, make tools.

Install XDC and tools (puppeth) (instructions in makefile)

Install docker, fetch latest docker image and rename as per docker-compose.

Create genesis.json from puppeth (./puppeth.sh)

Run setupnodes.sh (I usually just copy paste the commands from the file)
entry point from the dockerhub image did not allow for xdc-local to be run, 
so use empty entrypoint and use the binary directly.

will need to manually edit the bootnode enode address in the docker-compose.yml file from the logs of docker compose up
