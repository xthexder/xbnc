xbnc
====

xthexder's BNC - Custom IRC bouncer written in golang.

###status
Very buggy and not usable

###Setup
1. Open config.json and set the BNC listening host and port
2. Add a new user file in the users directory. Use test.json as template (this file should be deleted before use)
3. Add a new server in your IRC client with host and port as specified in config.json. You also need to add User Name. Set it to the same as Login in your user config.
4. Connect! The server will ask you to type in your password. This is what you set in the user config.
