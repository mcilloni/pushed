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
import groovy.transform.PackageScope
import groovy.util.logging.Log

/**
 * Parses Pushd configuration, and wraps access to it.
 */
@Log
final class Config {

    private static Config sInstance

    @PackageScope static Config read(String path) {
        sInstance = [path] as Config
    }

    static Config getValues() {
        sInstance
    }

    private final ConfigObject mConfig
    private final SocketAddress mSockAddr
    private final String mRedisHost
    private final Integer mRedisDb

    /**
     * Reads a config file in Groovy format, and checks if it is correctly formed (it must have a pushd root and must specify installation dir of pushd.
     * @param path a string containing the path to a config file
     * @throws ConfigException
     */
    private Config(String path) throws ConfigException {
        try {

            if (mConfig) {
                throw ["Config already initialized"] as ConfigException
            }

            log.info "Parsing $path with ConfigSlurper..."

            def conf = ([] as ConfigSlurper).parse(([path] as File).text)

            if(!conf.pushd) {
                throw ['no pushd root in config'] as ConfigException
            }

            conf = conf.pushd

            if(!conf.installPath) {
                throw ['no install path in config'] as ConfigException
            }

            def port = 8955

            if(conf.port) {
                try {
                    port = conf.port as Integer
                } catch (Exception ignore) {
                    throw ["${conf.port} is not a valid port number"] as ConfigException
                }
            }

            def host = 'localhost'

            if(conf.host) {
                try {
                    host = conf.host as String
                } catch (Exception ignore) {
                    throw ["${conf.host} is not a valid hostname"] as ConfigException
                }
            }

            mSockAddr = [host,port] as InetSocketAddress

            mRedisHost = 'localhost'

            if(conf.redisHost) {
                try {
                    mRedisHost = conf.redisHost as String
                } catch (Exception ignore) {
                    throw ["${conf.redisHost} is not a valid hostname"] as ConfigException
                }
            }

            mRedisDb = 0

            if(conf.redisDb) {
                try {
                    mRedisDb = conf.redisDb as Integer

                    if (!(mRedisDb in (0..15))) {
                        throw ["${conf.redisDb} is not in 0-15 range"] as ConfigException
                    }
                } catch (Exception ignore) {
                    throw ["${conf.redisDb} is not a valid integer value"] as ConfigException
                }
            }

            mConfig = conf

        } catch (Exception e) {
            throw [e] as ConfigException
        }

        log.info "No exceptions, $path is a valid pushd config."

    }

    SocketAddress getSocketAddress() {
        mSockAddr
    }

    String getRedisHost() {
        mRedisHost
    }

    Integer getRedisDb() {
        mRedisDb
    }

    /**
     * Returns a list of the List<Connector> specified by this config, initializing them if needed.
     * @return a list of the List<Connector> specified by this config
     */
    List<Connector> getConnectors() {
        if(!Connector.sConnectors) {
            Connector.loadConnectors(mConfig)
        }
        Connector.sConnectors
    }
}

@CompileStatic
class ConfigException extends Exception {

    ConfigException(Exception exc) {
        super(exc)
    }

    ConfigException(String msg) {
        super(msg)
    }

}