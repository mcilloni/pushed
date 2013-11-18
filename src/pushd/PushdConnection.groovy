/*
 * This file is part of ${PROJECT_NAME}.
 *
 *     ${PROJECT_NAME} is free software: you can redistribute it and/or modify
 *     it under the terms of the GNU General Public License as published by
 *     the Free Software Foundation, either version 3 of the License, or
 *     (at your option) any later version.
 *
 *     ${PROJECT_NAME} is distributed in the hope that it will be useful,
 *     but WITHOUT ANY WARRANTY; without even the implied warranty of
 *     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *     GNU General Public License for more details.
 *
 *     You should have received a copy of the GNU General Public License
 *     along with ${PROJECT_NAME}.  If not, see <http://www.gnu.org/licenses/>.
 *
 *     (C) 2013 Marco Cilloni <marco.cilloni@yahoo.com>
 */

package pushd

import groovy.json.JsonException
import groovy.json.JsonSlurper
import groovy.util.logging.Log

/**
 * Handles a connection with a client.
 */
@Log
class PushdConnection extends Thread {

    private BufferedReader mInput
    private BufferedWriter mOutput

    PushdConnection(BufferedReader input, BufferedWriter output) {
        (mInput,mOutput) = [input,output]
    }

    @Override
    void run() {
        try {
            while(true) {
                String line = this.mInput.readLine()
                Request req = [line] as Request
                switch (req.request) {
                    case Request.REGISTER:
                        if (!req.content.name){
                            throw ["No name given"] as PushdOperationException
                        }
                        registerUser req.content.name as String
                        break

                    case Request.PUSH:
                        /*TODO: implement logic */
                        break

                    case Request.LOAD:
                        if (!req.content.jar) {
                            throw ["No jar path given"] as PushdOperationException
                        }

                        Connector.load req.content.jar as String, req.content as Map
                        break

                    case Request.SUBSCRIBE:
                        if (!req.content.name){
                            throw ['No name given'] as PushdOperationException
                        }

                        String name = req.content.name

                        PushdUser user
                        if(!(user = PushDB.db.users[name])) {
                            throw ["no user $name found in database"] as PushdOperationException
                        }

                        user.subscriptions << name

                        break

                    default:
                        log.severe "Weird error, request $req.request has been allowed. Report bug."
                        break
                }
            }
        } catch(PushdRequestException | PushDBException | PushdOperationException e) {
            this.mOutput << error(e.localizedMessage)
        } finally {
            [mInput,mOutput].each { it.close() }
        }

    }

    private static void registerUser(String name) throws PushDBException, PushdOperationException {

        PushDB.db.users << name

    }

    private static String error(String message) {
        "{ \"error\" : \"$message\"}"
    }

}

class Response {

    static {
        Closure<String> format = (String.&format).curry '{ "response" : "%s" }'
        OK = format 'OK'
        MALFORMED = format 'MALFORMED'
    }

    final static String OK, MALFORMED

}

class Request {

    static {
        VALID_VALUES = ['REGISTER', 'PUSH', 'LOAD', 'SUBSCRIBE', 'UNSUBSCRIBE'].asImmutable()
        (REGISTER, PUSH, LOAD, SUBSCRIBE, UNSUBSCRIBE) = VALID_VALUES
    }

    final static String REGISTER, PUSH, LOAD, SUBSCRIBE, UNSUBSCRIBE
    final static List<String> VALID_VALUES

    private Map mObj
    private String mRequest

    Request(String text) throws PushdRequestException {
        try {
            def res = ([] as JsonSlurper).parseText text
            if (!(Map.isAssignableFrom(res.class))){
                throw ['Non-object json response'] as PushdRequestException
            }

            this.mObj = (res as Map).asImmutable()
            if (!(this.mRequest = this.mObj.request)) {
                throw ['Malformed request: invalid request field'] as PushdRequestException
            }

            this.mRequest = this.mRequest.trim().toUpperCase()
            if(!(this.mRequest in VALID_VALUES)) {
                throw ["Invalid request: ${this.mRequest}"] as PushdRequestException
            }

        } catch (JsonException ignore) {
            throw ['Malformed message received'] as PushdRequestException
        }
    }

    String getRequest() {
        this.mRequest
    }

    Map getContent() {
        this.mObj
    }

}

class PushdRequestException extends Exception {

    PushdRequestException(String string) {
        super(string)
    }

    PushdRequestException(Throwable t) {
        super(t)
    }

}

class PushdOperationException extends Exception {

    PushdOperationException(String string) {
        super(string)
    }

    PushdOperationException(Throwable t) {
        super(t)
    }

}