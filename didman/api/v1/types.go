/*
 * Copyright (C) 2021 Nuts community
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 */

package v1

import (
	"github.com/nuts-foundation/nuts-node/didman"
	"schneider.vip/problem"
)

// Error is an alias for the internally used problem.Problem
type Error = problem.Problem

// ContactInformation is an alias for the already defined didman.ContactInformation
type ContactInformation = didman.ContactInformation

// OrganizationSearchResult is an alias for the already defined didman.OrganizationSearchResult
type OrganizationSearchResult = didman.OrganizationSearchResult
