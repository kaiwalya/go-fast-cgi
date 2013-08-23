package main;

/*
when a connection is available, 
	Send connection to io service queue

when new connection in socket service queue
	Read or write and put back on cpu service queue
*/