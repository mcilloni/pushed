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

import groovy.transform.PackageScope
import groovy.transform.ToString
import groovy.util.logging.Log
import redis.clients.jedis.Jedis
import redis.clients.jedis.JedisPool
import redis.clients.jedis.JedisPoolConfig

import static pushd.Connector.isValidConnectorName

/**
 * DAO layer for pushd (uses and requires Redis)
 */
@Log
final class PushDB {

    private final static String sSystemPrefix = 'pushd:', sUsersPrefix = 'pushdusers:', sServices = sSystemPrefix + 'services', sSubscriptions = 'subscriptions'

    //static
    private static PushDB sInstance

    static PushDB connect() throws PushDBException {
        sInstance = [Config.values.redisHost] as PushDB
    }

    static PushDB getDb() {
        sInstance
    }

    //instance
    private JedisPool mConnectionPool

    private PushDB(String redisHost) throws PushDBException {
        this.mConnectionPool = [[] as JedisPoolConfig,redisHost]
        log.info 'Connected at redis.'

        Jedis jedis = this.jedis

        def type = jedis.type(sServices)

        if (!(type in ['none', 'list'])) {
            throw [sServices, 'none or list', type] as PushDBException
        }

    }

    PushdUser addUser(String name) throws PushDBException {
        def prefName = sUsersPrefix + name

        Jedis jedis = this.jedis
        if(!jedis.hsetnx(prefName,'name',name)) {
            throw ["can't add user $name: $prefName already present"] as PushDBException
        }

        jedis.hset(prefName, sSubscriptions,'')

        new PushdUserImpl(name)
    }

    Boolean existsUser(String name) {
        this.jedis.exists sUsersPrefix + name
    }

    PushdUserList getUsers() throws PushDBException {
        Jedis jedis = this.jedis
        def list = jedis.keys(sUsersPrefix+'*').collect { String username ->
            try {
                ((username  =~ ':(.+?)$')[0] as ArrayList<String>)[1] //I like type hinting in Idea
            } catch (IndexOutOfBoundsException ignore) {
                throw ["Invalid username found in Redis: $username"] as PushDBException
            }
        }

        Map uMap
        uMap = [
            getAt : { String name ->
                this.existsUser(name) ? new PushdUserImpl(name) : null
            },

            add : { String name ->
                this.addUser name
            },

            contains: { String name ->
              this.existsUser name
            },

            iterator : {
                Map impl //needed for runtime
                impl = [
                    lIter : list.iterator(),

                    hasNext : {
                        impl.lIter.hasNext()
                    },

                    next : {
                        new PushdUserImpl(impl.lIter.next() as String)
                    },

                    remove : {
                        throw ["Remove not implemented"] as UnsupportedOperationException
                    }
                ]

                impl as Iterator<PushdUser>
            }

        ]
        uMap as PushdUserList
    }

    PushdSubscriptions getServicesForUser(String user) {
        Jedis jedis = this.jedis
        def prefixUser = sUsersPrefix + user

        def subsString

        if ((subsString = jedis.hget(prefixUser, sSubscriptions)) == null || subsString.endsWith(':')) {
            throw ["user $user has malformed fields or is not existant"] as PushDBException
        }

        subsString = subsString.split(':') as LinkedList

        Map subs;
        subs = [

            haystack: subsString,

            contains: { String needle ->
                needle in subs.haystack
            },

            add: { String serviceName ->
                this.registerUserToService(user, serviceName)
                subs.haystack << serviceName
            },

            leftShift: { Connector connector ->
                subs.add connector.name
            },

            iterator: { subs.haystack.iterator() }
        ]
        subs as PushdSubscriptions
    }

    Boolean isUserRegisteredTo(String user, String serviceName)  throws PushDBException {
        serviceName in this.getServicesForUser(user)
    }

    void registerUserToService(String userName, String serviceName) throws PushDBException {

        if (!isValidConnectorName(serviceName)) {
            throw ["$serviceName is an invalid connector name"] as PushDBException
        }

        Jedis jedis = this.jedis

        String prefixName = sUsersPrefix + userName, services

        if(!(services = jedis.hget(prefixName,sSubscriptions)) || services.endsWith(':')) {
            throw ["unexistant user or corrupted $sSubscriptions field for $userName"] as PushDBException
        }

        services << serviceName

        jedis.hset prefixName,sSubscriptions,services

    }

    //gets a jedis from the pool, initialized correctly with db number and all the things we all love.
    @PackageScope Jedis getJedis() {
        def jedis = this.mConnectionPool.resource
        jedis.select Config.values.redisDb
        jedis
    }

    private void mapSerialize(String prefix, String identifier, Map map) throws PushDBException {
        identifier = prefix + identifier
        Jedis jedis = this.jedis
        def type = jedis.type identifier
        if (!(type in  ['hash','none'])) {
            throw [identifier,'hash',type] as PushDBException
        }

        jedis.hmset identifier,map
    }

    /**
     * Returns true if field is of one of the types in type. If type is null then the field is valid only if exists (type != none)
     * @param field
     * @param type
     * @return
     */
    private void checkFieldValid(String field, String... types) {

        Jedis jedis = this.jedis
        PushDBException exc;
        if (types) {
            if (!jedis.exists(field)) {
                exc = ["$field does not exist in database"]
            }
        } else {
            def type = jedis.type(field)
            if (!(type in types)) {
                exc = [field, { //unwrap parameters
                    StringBuffer buf = [types[0]]
                    types[1..-1].each { buf << ' or ' ; buf << it }

                    buf.toString()
                }(), type]
            }
        }

        if (exc) {
            throw exc
        }

    }

    private class PushdUserImpl implements PushdUser {

        private String mName

        PushdUserImpl(String userName) {
            this.mName = userName
        }

        @Override
        String getName() {
            this.mName
        }

        @Override
        PushdSubscriptions getSubscriptions() throws PushDBException {
            PushDB.this.getServicesForUser(this.mName)
        }
    }
}

interface PushdUserList extends Iterable<PushdUser> {
    PushdUser getAt(String name) throws PushDBException
    PushdUserList add(String name) throws PushDBException
    Boolean contains(String name) throws PushDBException
}

interface PushdUser {
    String getName()
    PushdSubscriptions getSubscriptions() throws PushDBException
}

interface PushdSubscriptions extends Iterable<String> {
    Boolean contains(String service) throws PushDBException
    void add(String service) throws PushDBException
    void leftShift(Connector connector) throws PushDBException
}

class PushDBException extends Exception {
    PushDBException(String string) {
        super(string)
    }

    PushDBException(String id, String expected, String found) {
        super("malformed identifier in Redis database: $id - Expected: $expected, Found: $found")
    }

}