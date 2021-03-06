/*
 * (C) Copyright 2016 Mycelius SA (http://mycelius.com/).
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * @fileOverview The definiton of Server class.
 * @author Dave Courtois
 * @version 1.0
 */

/**
 * The sever class is a representation of a distant server into a local object.
 * It regroup other manager object to give access to different services like file management, data manager etc.
 * @param {string} hostName The domain name of the server.
 * @param {string} ipv4 The ip v4 adress of the server.
 * @param {int} port The port number of the sever.
 * @constructor
 */
var Server = function (hostName, ipv4, port) {

    /**
     * @property {string} hostName The server side domain name.
     * @example www.cargoWebserver.com
     * @example localhost
     */
    this.hostName = hostName

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

    /**
     * @property {string} sessionId Identify the client with the server.
     */
    this.sessionId = null

    /**
     * @property {object} conn The web socket connection between the client and the server. 
     */
    this.conn = null

    /**
     * @property {LanguageManager} languageManager Manage the language in GUI. 
     */
    this.languageManager = new LanguageManager()
    this.languageManager.appendLanguageInfo(languageInfo)

    /**
     * @property {EventHandler} eventHandler Manage the network event. 
     */
    this.eventHandler = new EventHandler()


    return this;
}

/**
 * Initialyse the server connection.
 * @param {function} onConnectionOpenCallback That function is call when the connection is open and ready to be use.
 * @param {function} onConnectionCloseCallback That function is call when the connection is close with the server. 
 */
Server.prototype.init = function (onConnectionOpenCallback, onConnectionCloseCallback) {
    // Initialisation of the connection.
    this.conn = initConnection("ws://" + this.ipv4 + ":" + this.port.toString() + "/ws", onConnectionOpenCallback, onConnectionCloseCallback)
}

/**
 * That function is use to execute javascript function on the server side and get the result back
 * if there any.
 * @param {string} sessionId  The active session id.
 * @param {string} functionSrc  The source code of the function itself.
 * @param {array} functionParams  The list of function parameters.
 * @param {callback} progressCallback The function called when information is transfer from the server to client.
 * @param {callback} successCallback The success function to execute if the function success
 * @param {callback} errorCallback Call in case of error.
 * @param {object} caller This is the caller object of the function, it can be use full if other action are to be call on it.
 */
Server.prototype.executeJsFunction = function (functionSrc, functionParams, progressCallback, successCallback, errorCallback, caller) {
    var params = []
    params.push(createRpcData(functionSrc, "STRING", "functionSrc"))

    for (var i = 0; i < functionParams.length; i++) {
        params.push(functionParams[i])
    }

    var rqst = new Request(randomUUID(), this.conn, "ExecuteJsFunction", params,
        // Progress call back
        function (index, total, caller) {
            if (progressCallback != undefined) {
                progressCallback(index, total, caller)
            }
        },
        // success call back
        function (id, results, caller) {
            // Format the results.
            var result = results["result"]
            // Call the successCallback whit the information.
            if (successCallback != undefined) {
                successCallback(result, caller)
            }
            return true;
        },
        // Error call back
        function (errMsg, caller) {
            if (errorCallback != undefined) {
                errorCallback(errMsg, caller)
            }
        },
        caller
    );
    rqst.send()
}

/*
 * Handle the messages received.
 */
Server.prototype.handleMessage = function (conn, data) {

    // I will decode the message.
    var msg = RpcMessage.decode(data);

    // Here the sever ask for something.
    if (msg.type === 0) {
        // Request
        //console.log("request receive.");
        // I will create the Rpc request object.
        var request = new Request(msg.rqst.id, conn, msg.rqst.method, msg.rqst.params, null, null, null);
        request.process()
    }
    // Here I receive an answer from the server.
    else if (msg.type === 1) {
        // Response
        //console.log("Response received.");
        var response = new Response(msg.rsp.id, conn, msg.rsp.results, null, null, null);
        if (pendingMessage[response.id] != undefined) {
            processPendingMessage(response.id)
        } else {
            var rqst = pendingRequest[response.id]
            if (rqst != undefined) {
                delete pendingRequest[rqst.id];
                response.execute(rqst);
            }
        }

    }
    // I receive an error from the server.
    else if (msg.type === 2) {
        // error
        //console.log("Error received.");
        // Create the error message.
        var err = new ErrorMsg(msg.err.id, conn, msg.err.code, msg.err.message, msg.err.data, null, null, null)
        err.catch()

    }
    // I receive event from the server.
    else if (msg.type === 3) {
        // event
        //console.log("Event received.");
        var evt = new EventMsg(msg.evt.id, conn, msg.evt.name, msg.evt.code, msg.evt.evtData, null, null, null)
        this.eventHandler.broadcastLocalEvent(evt)
    }
    // This is a transfer message use whit chunk message.
    else if (msg.type === 4) {

        // Here I need to save the message into the pending message map.
        if (pendingMessage[msg.id] == undefined) {
            // I will create the array first.
            pendingMessage[msg.id] = new Array()
        }

        // Now I will save the message.
        pendingMessage[msg.id][msg.index] = msg

        if (pendingRequest[msg.id] != undefined) {
            if (pendingRequest[msg.id].progressCallback != null) {
                pendingRequest[msg.id].progressCallback(msg.index, msg.total, pendingRequest[msg.id].caller)
            }
        }

        // I will now create the answer for that transfer message and send it back
        // to the server.
        var results = new Array()
        var resp = new Response(msg.id, conn, results, null, null, null)
        resp.send()

        // If the message is fully transfer.
        if (msg.index == msg.total - 1) {
            // Here is the case of fragmented response. so I need to reconstruct the response from
            // the data contain in the assembled message.
            var data = []
            for (var i = 0; i < pendingMessage[msg.id].length; i++) {
                var begin = pendingMessage[msg.id][i].data.offset;
                var end = pendingMessage[msg.id][i].data.limit;
                data[i] = pendingMessage[msg.id][i].data.view.subarray(begin, end)
            }

            // Now I will delete the message.
            delete pendingMessage[msg.id]

            // Here i will made use of file data to read a messge...
            var fileReader = new FileReader();
            fileReader.onload = function (server) {
                return function () {
                    if (this.result != null) {
                        var arrayBuffer = this.result;
                        server.handleMessage(server.conn, arrayBuffer);
                    } else {
                        console.log("File data cannot be read!!!")
                    }
                }
            }(this);

            fileReader.readAsArrayBuffer(new Blob(data));
        }
    }
};

/**
 * Get the current session id.
 */
Server.prototype.setSessionId = function (initCallback) {
    var params = new Array();
    // Register this listener to the server.
    var rqst = new Request(randomUUID(), this.conn, "GetSessionId", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, results, caller) {
            // Keep the session id...
            caller.server.sessionId = results.result

            // Each application must contain a main.
            if (main != null) {
                if (caller.initCallback != undefined) {
                    caller.initCallback()
                }
            } else {
                // I will show the project manager page.
                // TODOO error 404
                //alert("Error 404, Cargo handler.js line 121")
            }
        },
        // Error callback...
        function () {

        }, { "server": this, "initCallback": initCallback });
    rqst.send();
}

/**
 * Test if a server is reachable. Receive Pong as answer.
 */
Server.prototype.ping = function (successCallback, errorCallback, caller) {
    var params = new Array();

    // Register this listener to the server.
    var rqst = new Request(randomUUID(), this.conn, "Ping", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, result, caller) {
            // Keep the session id...
            if (caller.successCallback != null) {
                caller.successCallback(result, caller.caller)
            }
        },
        // Error callback...
        function (errObj, caller) {
            if (caller.errorCallback != null) {
                caller.errorCallback(errObj, caller.caller)
            }
        }, { "successCallback": successCallback, "errorCallback": errorCallback, "caller": caller });
    rqst.send();
}

/**
 * Run a visual basic script with a given name on the server side.
 */
Server.prototype.executeVbSrcript = function (scriptName, args, successCallback, errorCallback, caller) {
    var params = new Array();
    params.push(createRpcData(scriptName, "STRING", "scriptName"))
    params.push(createRpcData(args, "JSON_STR", "args"))

    // Register this listener to the server.
    var rqst = new Request(randomUUID(), this.conn, "ExecuteVbScript", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, results, caller) {
            // Keep the session id...
            caller.successCallback(results, caller.caller)
        },
        // Error callback...
        function (errorMsg, caller) {
            caller.errorCallback(errorMsg, caller.caller)
        }, { "successCallback": successCallback, "errorCallback": errorCallback, "caller": caller });
    rqst.send();
}


/**
 * Run and executable command on the server and get the results.
 */
Server.prototype.runCmd = function (name, args, successCallback, errorCallback, caller) {
    var params = new Array();
    params.push(createRpcData(name, "STRING", "name"))
    params.push(createRpcData(args, "JSON_STR", "args"))

    // Register this listener to the server.
    var rqst = new Request(randomUUID(), this.conn, "RunCmd", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, results, caller) {
            // Keep the session id...
            caller.successCallback(results, caller.caller)
        },
        // Error callback...
        function (errorMsg, caller) {
            caller.errorCallback(errorMsg, caller.caller)
        }, { "successCallback": successCallback, "errorCallback": errorCallback, "caller": caller });
    rqst.send();
}

/**
 * Get the list of services and their respective source code. The code
 * permit to get access to service remote actions.
 * @param {function} successCallback The function is call in case of success and the result parameter contain objects we looking for.
 * @param {function} errorCallback In case of error.
 * @param {object} caller A place to store object from the request context and get it back from the response context.
 */
Server.prototype.getServicesClientCode = function (successCallback, errorCallback, caller) {
    // server is the client side singleton.
    var params = new Array();

    // Register this listener to the server.
    var rqst = new Request(randomUUID(), this.conn, "GetServicesClientCode", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, results, caller) {
            // Keep the session id...
            caller.successCallback(results["result"], caller.caller)
        },
        // Error callback...
        function (errorMsg, caller) {
            caller.errorCallback(errorMsg, caller.caller)
        }, { "successCallback": successCallback, "errorCallback": errorCallback, "caller": caller });
    rqst.send();
}

/**
 * Close the server.
 */
Server.prototype.stop = function (successCallback, errorCallback, caller) {
    // server is the client side singleton...
    var params = []

    var rqst = new Request(randomUUID(), this.conn, "Stop", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, results, caller) {
            // Keep the session id...
            caller.successCallback(results["result"], caller.caller)
        },
        // Error callback...
        function (errorMsg, caller) {
            caller.errorCallback(errorMsg, caller.caller)
        }, { "successCallback": successCallback, "errorCallback": errorCallback, "caller": caller });

    rqst.send();
}


/**
 * Get the server root path. This is where the WebApp directory is located on the sever.
 */
Server.prototype.getRootPath = function (successCallback, errorCallback, caller) {
    // server is the client side singleton...
    var params = new Array();

    var rqst = new Request(randomUUID(), this.conn, "GetRootPath", params,
        // Progress callback
        function () { },
        // Success callback
        function (id, results, caller) {
            // Keep the session id...
            caller.successCallback(results["result"], caller.caller)
        },
        // Error callback...
        function (errorMsg, caller) {
            caller.errorCallback(errorMsg, caller.caller)
        }, { "successCallback": successCallback, "errorCallback": errorCallback, "caller": caller });

    rqst.send();
}

/**
* This little structure is use to keep the cc information.
* @param name  Just a string can be John Doe, mr. Doe etc.
* @param email  The email of the cc.
* @constructor
*/
var CarbonCopy = function (name, email) {
    // Hint about type name.
    this.TYPENAME = "Server.CarbonCopy"

    /**
     * @property {string} Name The name of the carbon copy.
     */
    this.Name = name

    /**
     * @property {string} Mail The email addresse of the carbon copy.
     */
    this.Mail = email

    return this
}


/**
 * Create a new empty query.
 * @constructor
 */
var EntityQuery = function () {
    this.TYPENAME = "Server.EntityQuery"

    // The name of the entity
    this.TypeName = ""
    
	// The list of field to retreive, delete or modify
    this.Fields = []
    
	// The base index, this must be of form indexFieldName=indexFieldValue
    this.Indexs = []
    
	// The query to execute by the search engine.
    this.Query = ""
    
    return this
}

/**
 * This structure is use in the transfer of files use in attach.
 * @param {string} fileName  The name of the file
 * @param {string} fileData  The file data
 * @constructor
 */
var Attachment = function (fileName, fileData) {
    this.TYPENAME = "Server.Attachment"
    /**
     * @property {string} FileName the name of the file attachement. It can also contain the file path on the server side,
     * if is the case the file data will be take from this file.
     */
    this.FileName = fileName
    /**
     * @property {string} FileData The data of the file. If it's null the file data will be taked from the server side.
     */
    this.FileData = fileData
    return this
}

/**
 * Node js import function.
 */
function require(moduleId) {

}