/*
 * This file is part of pushd.
 *
 *     pushd is free software: you can redistribute it and/or modify
 *     it under the terms of the GNU General Public License as published by
 *     the Free Software Foundation, either version 3 of the License, or
 *     (at your option) any later version.
 *
 *     pushd is distributed in the hope that it will be useful,
 *     but WITHOUT ANY WARRANTY; without even the implied warranty of
 *     MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *     GNU General Public License for more details.
 *
 *     You should have received a copy of the GNU General Public License
 *     along with pushd.  If not, see <http://www.gnu.org/licenses/>.
 *
 *     (C) 2013 Marco Cilloni <marco.cilloni@yahoo.com>
 */

package pushd

import groovy.transform.CompileStatic
import groovy.util.logging.Log

import java.util.concurrent.atomic.AtomicBoolean

/**
 * Listens for incoming requests and dispatches them to Connectors if needed.
 */
@Log
@CompileStatic
class PushdListener extends Thread {

    private AtomicBoolean mExecute

    private ServerSocket mServerSocket

    private List<PushdConnection> mConnections

    PushdListener() {
        super()
        this.mServerSocket = []
        this.mServerSocket.bind Config.values.socketAddress
        this.mExecute = [true]
        this.mConnections = [].asSynchronized()

    }

    void terminate() {
        this.mExecute.set(false)
        this.mServerSocket.close()
    }

    @Override
    void run() {
        try {
            this.mServerSocket.accept { Socket client ->
                client.withStreams { InputStream input, OutputStream output ->
                        this.mConnections << ([[[input] as InputStreamReader] as BufferedReader, [[output] as OutputStreamWriter] as BufferedWriter] as PushdConnection)
                }
            }
        } catch (SocketException ignore) {} //usually thrown because of terminate() and close()
    }

}
