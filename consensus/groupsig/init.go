//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package groupsig

const (
	// Curve254 is a 256 - bit curve
	Curve254 = 0

	// Curve382_1 is 384-bit curve 1
	Curve382_1 = 1

	// Curve382_2 is 384-bit curve 2
	Curve382_2 = 2

	// DefaultCurve is default curve used
	DefaultCurve = 1

	/*
	   Default curve related parameters start, If the number of digits in the default curve is adjusted,
	   these parameters also need to be modified.
	*/

	// IDLENGTH is ID byte length (256 bits, same as private key length)
	IDLENGTH = 32

	// PUBKEYLENGTH is public key byte length (1024 bits)
	PUBKEYLENGTH = 128

	// SECKEYLENGTH is private key byte length (256 bits)
	SECKEYLENGTH = 32

	// SIGNLENGTH is signature byte length (256 bits + 1 byte parity)
	SIGNLENGTH = 33

	/*
	   Default curve related parameters end
	*/

	// HASHLENGTH is hash byte length (golang.sha3, 256 bits. Same as the common package)
	HASHLENGTH = 32 //
)
