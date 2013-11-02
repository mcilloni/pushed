package pushd.tests
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

pushd {

    installPath = '/usr/lib/pushd'

    //default port is 8955
    //port = 8955

    //default host is localhost
    //host = 'localhost'

    //default redis host is localhost
    //redisHost = 'localhost'

    //facultative: pushd will run if no connector is installed, but this will surely get weird.
    connectorsPath = "/home/marco/Workspace/Pushd/out/production/Pushd/"

    connectors {
        //gcm could be any name, even cuddlyTinyBear. It's here as a mnemonic and I won't judge you.
        example {
            jarname = 'ex.jar'
            settings {
                token = ""
            }
        }
    }

}