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

import java.lang.reflect.Constructor
import java.util.jar.Attributes
import java.util.jar.JarFile

/**
 * This class represents a connector. Any connector is associated with a service, and gets its configuration from
 * a ConfigObject retrieved by Config.
 */
@Log
abstract class Connector {

    final static Integer TIMEOUT_SECONDS = 1

    private String mName
    private String mVersion
    private Status mCurrentStatus

    @PackageScope static Set<Connector> sConnectors

    @PackageScope Closure problemReport

    static {
        sConnectors = ([] as Set<Connector>).asSynchronized() //must be syncronized
    }

    static Connector load(String jarPath, ConfigObject connectorConfig) throws ConnectorException,IOException {

        //making this absolute
        jarPath = ([jarPath] as File).absolutePath

        log.info "Loading Jar File $jarPath as pushd Connector"

        def manifest = ([jarPath] as JarFile).manifest

        def properties = [:] as LinkedHashMap<String,String>

        //check if the connector specifies all required classes.
        ['name', 'version', 'class', 'flatten'].each { String key ->

            def fullKey = (["Pushd-Connector-${key.capitalize()}"] as Attributes.Name) as Object

            if(!(fullKey in manifest.mainAttributes)) {
                throw new ConnectorException(jarPath, "Pushd-Connector-${key.capitalize()}")
            }

            properties << [(key):manifest.mainAttributes[fullKey] as String]

        }

        if (!(properties['flatten'] in ['true', 'false'])) {
            throw ["Not boolean value found for Pushd-Connector-Flatten: ${properties['flatten']}"] as ConnectorException
        }

        log.info "$jarPath has a valid Connector manifest, trying to initialize it..."

        //load the Connector into the classLoader
        this.classLoader.rootLoader.addURL (["file://$jarPath"] as URL)

        Class<? extends Connector> connectorClass

        try {
            connectorClass = Class.forName(properties['class']) as Class<? extends Connector>
        } catch (ClassNotFoundException ignore) {
            throw ["Class ${properties['class']} has not been found in $jarPath"] as ConnectorException
        }

        if (!Connector.isAssignableFrom(connectorClass)) {
            throw ["${properties['class']} does not extend Connector"] as ConnectorException
        }

        Constructor<Connector> constructor = null

        if (!connectorClass.constructors.any { Constructor<Connector> ctor ->
            constructor = ctor
            ctor.parameterTypes*.name as Set == ['java.lang.String','java.lang.String','java.util.Map'] as Set
        }) {
            throw ["No valid constructor declared by ${properties['class']} for connector ${properties['name']}}"] as ConnectorException
        }

        Connector connector = constructor.newInstance(properties['name'], properties['version'], (properties['flatten'] == 'false') ? connectorConfig : connectorConfig.flatten([:]))

        log.info "Anything went perfectly and ${properties['name']} has been correctly initialized"

        connector

    }

    /**
     * This constructor should be overridden equally by each connector, or it will be impossibile for the load method to invoke it.
     * @param name The name of the connector
     * @param version The connector version
     * @param config The connector's config. This is actually a ConfigObject, but it's not specified for compatibility with java. If you want a real map set Pushd-Connector-Flatten in the manifest to true
     * @throws ConnectorException
     */
    protected Connector(String name, String version, Map config) throws ConnectorException {
        (mName, mVersion, mCurrentStatus) = [name, version, Status.OFF]
        sConnectors << this
    }

    String getName() {
        this.mName
    }

    String getVersion() {
        this.mVersion
    }

    Status getStatus() {
        this.mCurrentStatus
    }

    protected setStatus(Status status) {
        this.mCurrentStatus = status
    }

    /**
     * This method is called by pushd at the moment it loads, and must not block the program flow, or it will be killed.
     * Timeout is of 1 second. If you need more time, dispatch a thread and do the things you need to do.
     */
    abstract void init()

    @PackageScope void initConnector() {

    }

    /**
     * This method is called by pushd at the moment it loads.
     * Timeout is of 1 second. If you need more time, dispatch a thread and do the things you need to do.
     * Remember to override and run it or you will probably encounter errors.
     */
    void destroy() {
        sConnectors -= this
    }

    @CompileStatic
    static class ConnectorException extends Exception {

        ConnectorException(Exception e) {
            super(e)
        }

        ConnectorException(String message) {
            super(message)
        }

        ConnectorException(String jarPath, String missingProperty) {
            super("jar file $jarPath does not specify the $missingProperty field in its manifest")
        }

    }

    @CompileStatic
    static enum Status {

        //The connector is up and running.
        READY,

        //The connector is still busy initializing itself.
        BUSY,

        //The connector has stopped working.
        DEAD,

        //The connector has been suspended.
        SUSPENDED,

        //The connector has not been initialized yet.
        OFF
    }

}