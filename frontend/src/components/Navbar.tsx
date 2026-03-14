"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

interface AuthUser {
  name: string;
  role: string;
}

export default function Navbar() {
  const [user, setUser] = useState<AuthUser | null>(null);
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    try {
      const token = localStorage.getItem("token");
      const stored = localStorage.getItem("user");
      if (token && stored) {
        setUser(JSON.parse(stored));
      }
    } catch {
      // Ignore parse errors
    }
  }, []);

  function handleLogout() {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    setUser(null);
    window.location.href = "/";
  }

  return (
    <nav className="bg-gray-900 text-white shadow-lg">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-16 items-center justify-between">
          {/* Logo and primary nav */}
          <div className="flex items-center gap-8">
            <Link href="/" className="text-xl font-bold tracking-tight">
              BookQGen
            </Link>

            <div className="hidden items-center gap-6 sm:flex">
              <Link
                href="/"
                className="text-sm font-medium text-gray-300 transition-colors hover:text-white"
              >
                Home
              </Link>
              <Link
                href="/dashboard"
                className="text-sm font-medium text-gray-300 transition-colors hover:text-white"
              >
                Dashboard
              </Link>
              {user?.role === "admin" && (
                <Link
                  href="/admin"
                  className="text-sm font-medium text-gray-300 transition-colors hover:text-white"
                >
                  Admin
                </Link>
              )}
            </div>
          </div>

          {/* User section (desktop) */}
          <div className="hidden items-center gap-4 sm:flex">
            {user ? (
              <>
                <span className="text-sm text-gray-300">{user.name}</span>
                <button
                  onClick={handleLogout}
                  className="rounded-md bg-gray-700 px-3 py-1.5 text-sm font-medium transition-colors hover:bg-gray-600"
                >
                  Logout
                </button>
              </>
            ) : (
              <Link
                href="/login"
                className="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium transition-colors hover:bg-blue-500"
              >
                Sign In
              </Link>
            )}
          </div>

          {/* Mobile menu button */}
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="inline-flex items-center justify-center rounded-md p-2 text-gray-400 hover:bg-gray-800 hover:text-white sm:hidden"
            aria-label="Toggle menu"
          >
            <svg
              className="h-6 w-6"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
            >
              {menuOpen ? (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M6 18L18 6M6 6l12 12"
                />
              ) : (
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5"
                />
              )}
            </svg>
          </button>
        </div>
      </div>

      {/* Mobile menu */}
      {menuOpen && (
        <div className="border-t border-gray-700 sm:hidden">
          <div className="space-y-1 px-4 pb-3 pt-2">
            <Link
              href="/"
              className="block rounded-md px-3 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white"
              onClick={() => setMenuOpen(false)}
            >
              Home
            </Link>
            <Link
              href="/dashboard"
              className="block rounded-md px-3 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white"
              onClick={() => setMenuOpen(false)}
            >
              Dashboard
            </Link>
            {user?.role === "admin" && (
              <Link
                href="/admin"
                className="block rounded-md px-3 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white"
                onClick={() => setMenuOpen(false)}
              >
                Admin
              </Link>
            )}
            <div className="border-t border-gray-700 pt-2">
              {user ? (
                <>
                  <span className="block px-3 py-2 text-sm text-gray-400">
                    {user.name}
                  </span>
                  <button
                    onClick={handleLogout}
                    className="block w-full rounded-md px-3 py-2 text-left text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white"
                  >
                    Logout
                  </button>
                </>
              ) : (
                <Link
                  href="/login"
                  className="block rounded-md px-3 py-2 text-sm font-medium text-gray-300 hover:bg-gray-800 hover:text-white"
                  onClick={() => setMenuOpen(false)}
                >
                  Sign In
                </Link>
              )}
            </div>
          </div>
        </div>
      )}
    </nav>
  );
}
