#!/bin/sh

LINODE_API_KEY="rY0CbkBA83OGLDYW9t965qbp9ooc39AyUhGG6ZAM3SMjhUDccpgNET6eFs5PdgC2"
LINODE_DOMAIN_ID="179075"
LINODE_RESOURCE_ID="2438303"

wget "https://api.linode.com/?api_key=${LINODE_API_KEY}&api_action=domain.resource.update&DomainID=${LINODE_DOMAIN_ID}&ResourceID=${LINODE_RESOURCE_ID}&Target=[remote_addr]" -O - &> /dev/null
