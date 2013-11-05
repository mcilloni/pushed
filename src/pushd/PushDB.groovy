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

import groovy.grape.Grape
import groovy.util.logging.Log
import redis.clients.jedis.Jedis
import redis.clients.jedis.JedisPool
import redis.clients.jedis.JedisPoolConfig

/**
 * DAO layer for pushd (uses and requires Redis)
 */
@Grapes([
    @Grab(group='redis.clients',module='jedis',version='[2.2.1,)'), //Fetch jedis redis connector.
])
@Log
final class PushDB {

    private final static String sSystemPrefix = 'pushd:', sUsersPrefix = 'pushdusers:'

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
    }

    PushdUser addUser(String name) throws PushDBException {
        def prefName = sUsersPrefix + name

        Jedis jedis = this.jedis
        if(!jedis.hsetnx(prefName,'name',name)) {
            throw ["can't add user $name: $prefName already present"] as PushDBException
        }

        jedis.hset(prefName, 'subscriptions','')

        new PushdUserImpl(name)
    }

    PushdUserList getUsers() throws PushDBException {
        Jedis jedis = this.jedis
        def list = jedis.keys(sUsersPrefix+'*').collect { String username ->
            try {
                ((username  =~ ':(.*)')[0] as ArrayList<String>)[1] //I like type hinting in Idea
            } catch (IndexOutOfBoundsException ignore) {
                throw ["Invalid username found in Redis: $username"] as PushDBException
            }
        }
    }

    void registerService(String name) throws PushDBException {


    }

    //gets a jedis from the pool, initialized correctly with dbname and all the things we all love.
    private Jedis getJedis() {
        def jedis = this.mConnectionPool.resource
        jedis.select Config.values.redisDb
        jedis
    }

    private void mapSerialize(String prefix, String identifier,Map map) throws PushDBException {
        identifier = prefix + identifier
        Jedis jedis = this.jedis
        map.each { String key, String value ->
            def type = jedis.type(identifier)
            if (type == 'hash') {
                jedis.hset identifier,key,value
            } else {
                throw [identifier,'hash',type] as PushDBException
            }
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
        Map<String, String> getConnectorSettings() throws PushDBException {
            return null
        }

        @Override
        PushdSubscriptions getSubscriptions() throws PushDBException {
            return null
        }
    }
}

interface PushdUserList extends Iterable<PushdUser> {
    PushdUser getAt(String name)
    PushdUserList add(String name)
}

interface PushdUser {
    String getName()
    Map<String,String> getConnectorSettings() throws PushDBException
    PushdSubscriptions getSubscriptions() throws PushDBException
}

interface PushdSubscriptions extends Iterable<String> {
    Boolean contains(String service)
    void add(String service)
    void leftShift(Connector connector)
}

class PushDBException extends Exception {
    PushDBException(String string) {
        super(string)
    }

    PushDBException(String id, String expected, String found) {
        super("malformed identifier in Redis database: $id - Expected: $expected, Found: $found")
    }

}