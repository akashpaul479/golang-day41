# golang-day41

#JWT Authentication in Go

This project implements **jwt-based authentication** in Go.
It provides secure login , refresh , logout handlers and middleware for protecting routes with access and refesh tokens stored in cookies.

## Features
- Login with email/password
-Issue **Access Token** (15 minutes TTL) and **refresh token** (7 days TTL)
-refresh endpoint to renew access tokens 
-middleware to project routes
-logout handler to clear cookies
-secure cookie handling ('HTTP-only','samesite')

## Dependencies
-[github.com/golang-jwt/jwt/v5]
Install with: ```bash
go get github.com/golang-jwt/jwt/v5


