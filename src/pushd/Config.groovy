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
import pushd.Connector.Status

/**
 * Parses Pushd configuration, and wraps access to it.
 */
@Log
final class Config {

    private static ConfigObject sConfig
    private static SocketAddress sSockAddr
    private static String sRedisHost

    /**
     * Reads a config file in Groovy format, and checks if it is correctly formed (it must have a pushd root and must specify installation dir of pushd.
     * @param path a string containing the path to a config file
     * @throws ConfigException
     */
    static void read(String path) throws ConfigException {
        try {

            log.info "Parsing $path with ConfigSlurper..."

            def conf = new ConfigSlurper().parse new File(path).text

            if(!conf.pushd) {
                throw new ConfigException('no pushd root in config')
            }

            conf = conf.pushd

            if(!conf.installPath) {
                throw new ConfigException('no install path in config')
            }

            def port = 8955

            if(conf.port) {
                try {
                    port = conf.port as Integer
                } catch (Exception ignore) {
                    throw new ConfigException("${conf.port} is not a valid port number")
                }
            }

            def host = 'localhost'

            if(conf.host) {
                try {
                    host = conf.host as String
                } catch (Exception ignore) {
                    throw new ConfigException("${conf.host} is not a valid hostname")
                }
            }

            sSockAddr = [host,port] as InetSocketAddress

            sRedisHost = 'localhost'

            if(conf.redisHost) {
                try {
                    sRedisHost = conf.redisHost as String
                } catch (Exception ignore) {
                    throw new ConfigException("${conf.host} is not a valid hostname")
                }
            }

            sConfig = conf

        } catch (Exception e) {
            throw new ConfigException(e)
        }

        log.info "No exceptions, $path is a valid pushd config."

    }

    static SocketAddress getSocketAddress() {
        sSockAddr
    }

    static String getRedisHost() {
        sRedisHost
    }

    /**
     * Returns a list of the List<Connector> specified by this config, initializing them if needed.
     * @return a list of the List<Connector> specified by this config
     */
    static List<Connector> getConnectors() {
        if(!Connector.sConnectors) {
            loadConnectors()
        }
        Connector.sConnectors
    }

    private static List<Connector> loadConnectors() throws ConfigException, Connector.ConnectorException {
        if(!(sConfig.connectorsPath && sConfig.connectors)) {
            log.warning "no connectors/connectorsPath specified in config. Pushd will load but will be useless until you manually load one from the console. Please check your configuration"
            return []
        }

        File conDir = [sConfig.connectorsPath as String]

        if(!(conDir.exists() && conDir.directory )) {
            throw ["connectorsPath unexisting or not directory: ${conDir.name}"] as ConfigException
        }

        if(!(sConfig.connectors instanceof ConfigObject)) {
            throw ['connectors is not a config object in current config'] as ConfigException
        }

        log.info "Now loading specified connectors from ${conDir.absolutePath}"

        sConfig.connectors.each { String key, ConfigObject value ->

            if (!(value.jarname && value.jarname instanceof String)) {
                throw ["No or invalid jarname provided for $key"] as ConfigException
            }

            ConfigObject config = value.settings ?: null

            def connector = Connector.load(value.jarname as String, config)

            connector.problemReport = Config.&problemReport

            log.info "Loaded plugin ${connector.name}"
        }

        connectors

    }

    private static void problemReport(Connector connector, String message) {
        switch (connector.status) {
            case Status.DEAD:
                log.severe "Connector ${connector.name} has stopped working with reason $message"
                connector.destroy()
                break

            case Status.BUSY:
                log.warning "Connector ${connector.name} has still not fully initialized: $message"
                break

            default:
                log.warning "Connector ${connector.name} is reporting problems but is on status ${connector.status}. This is unsupported"
        }

    }

    @CompileStatic
    static class ConfigException extends Exception {

        ConfigException(Exception exc) {
            super(exc)
        }

        ConfigException(String msg) {
            super(msg)
        }

    }

}
