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


@Grab(group='commons-daemon', module='commons-daemon', version='1.0.15')
import org.apache.commons.daemon.Daemon
import org.apache.commons.daemon.DaemonContext
import org.apache.commons.daemon.DaemonInitException
import pushd.Config.ConfigException

/**
 * Pushd main daemon.
 */
@Log
@CompileStatic
final class Pushd implements Daemon {

    private PushdListener mListener

    @Override
    void init(DaemonContext daemonContext) throws DaemonInitException, Exception {

        log.info 'Initializing pushd daemon.'

        if (daemonContext.arguments.length != 1) {
            throw ["Wrong number of parameters: ${daemonContext.arguments.length}"] as DaemonInitException
        }

        try {
            //read config
            Config.read daemonContext.arguments[0]
        } catch (ConfigException e) {
            throw ["Problems while reading config file ${daemonContext.arguments[0]}: $e.localizedMessage"] as DaemonInitException
        }

    }

    @Override
    void start() throws Exception {
        //Start all connectors
        Config.connectors*.init()
        this.mListener = []
        this.mListener.run()
    }

    @Override
    void stop() throws Exception {
        this.mListener.terminate()
    }

    @Override
    void destroy() {
        //To change body of implemented methods use File | Settings | File Templates.
    }
}
