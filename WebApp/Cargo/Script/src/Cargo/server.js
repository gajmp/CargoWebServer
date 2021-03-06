// The default session id.

/**
 * Connection class
 * Contain information about the connection.
 */
var Connection = function(ipv4, port){
	/**
	 * @property {string} The id will be set at the connection time.
	 */
	this.id = ""; 
	
    /**
     * @property {string} ipv4 the ip adress of the server.
     * @example 127.0.0.1
     */
    this.ipv4 = ipv4

    /**
     * @property {int} port the port number.
     * @example 8080
     */
    this.port = port

	return this;
}

/**
 * Contain the informations use by executeJsFunction function parameters.
 */
var RpcData = function(values){
	this.name = values.name
	this.type = values.type
	this.dataBytes = values.dataBytes
	this.typeName = values.typeName
}

/**
 * The sever class is a representation of a distant server into a local object.
 */
 var Server = function (hostName, ipv4, port) {
     /**
     * @property {string} hostName The server side domain name.
     * @example www.cargoWebserver.com
     * @example localhost
     */
    this.hostName = hostName

	 /**
     * @property {object} The network connection.
     */
	this.conn = new Connection(ipv4, port)
	
    /**
     * @property {string} sessionId Identify the client with the server.
     */
    this.sessionId = sessionId; // sessionId is global.
 
	return this
 }
 
 /*
  * Open the connection with the other peer.
  */
 Server.prototype.init = function (onOpenConnectionCallback, onCloseConnectionCallback, caller) {
	// Set the connection callback.
	this.conn.onopen = onOpenConnectionCallback;
	this.conn.onclose = onCloseConnectionCallback;
	
	// initialise the connection.
	initConnection(sessionId, this, caller);
 }
 
 /*
 * Ping a given sever.
 */
 Server.prototype.ping = function (successCallback, errorCallback, caller) {
	ping(successCallback, errorCallback, caller, this.conn.id);
 }
 
/*
 * Execute JavaScript function.
 */
 Server.prototype.executeJsFunction = function (functionSrc, functionParams, progressCallback, successCallback, errorCallback, caller) {
	executeJsFunction(functionSrc, functionParams, progressCallback, successCallback, errorCallback, caller, this.conn.id);
 }

/**
 * Run a visual basic script with a given name on the server side.
 */
Server.prototype.executeVbSrcript = function (scriptName, args, successCallback, errorCallback, caller) {
	executeVbSrcript(scriptName, args, successCallback, errorCallback, caller, this.conn.id);
}

/**
 * Run and executable command on the server and get the results.
 */
Server.prototype.runCmd = function (name, args, successCallback, errorCallback, caller) {
	runCmd(name, args, successCallback, errorCallback, caller, this.conn.id);
}

/**
 * Get the list of services and their respective source code. The code
 * permit to get access to service remote actions.
 * @param {function} successCallback The function is call in case of success and the result parameter contain objects we looking for.
 * @param {function} errorCallback In case of error.
 * @param {object} caller A place to store object from the request context and get it back from the response context.
 */
Server.prototype.getServicesClientCode = function (successCallback, errorCallback, caller) {
    getServicesClientCode(successCallback, errorCallback, caller, this.conn.id);
}

/**
 * Close the server.
 */
Server.prototype.stop = function (successCallback, errorCallback, caller) {
    // server is the client side singleton...
    stop(successCallback, errorCallback, caller, this.conn.id);
}

/**
 * Create the local server object.
 */
var server = new Server("localhost", "127.0.0.1", 9393);

// Export class.
exports.server = server;
exports.Server = Server;
exports.RpcData = RpcData;
exports.Connection = Connection;

