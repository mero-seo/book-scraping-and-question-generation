"use client";

import { useState, useEffect } from "react";
import { listUsers, updateUserRole } from "@/lib/api";
import type { User } from "@/lib/types";
import Pagination from "@/components/Pagination";

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadUsers();
  }, [page]);

  async function loadUsers() {
    setLoading(true);
    try {
      const res = await listUsers({ page, limit: 20 });
      setUsers(res.data || []);
      setTotalPages(res.pagination.total_pages);
    } catch {
      setUsers([]);
    } finally {
      setLoading(false);
    }
  }

  async function toggleRole(user: User) {
    const newRole = user.role === "admin" ? "student" : "admin";
    if (!confirm(`Change ${user.name}'s role to ${newRole}?`)) return;
    try {
      await updateUserRole(user.id, newRole);
      loadUsers();
    } catch (err) {
      alert(err instanceof Error ? err.message : "Failed to update role");
    }
  }

  if (loading) return <div className="text-center py-12">Loading...</div>;

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">User Management</h1>

      <div className="space-y-3">
        {users.map((user) => (
          <div
            key={user.id}
            className="bg-white rounded-lg shadow p-4 flex justify-between items-center"
          >
            <div>
              <h3 className="font-medium">{user.name}</h3>
              <p className="text-sm text-gray-500">{user.email}</p>
              <div className="flex gap-2 mt-1">
                <span
                  className={`text-xs px-2 py-0.5 rounded ${
                    user.role === "admin"
                      ? "bg-purple-100 text-purple-700"
                      : "bg-blue-100 text-blue-700"
                  }`}
                >
                  {user.role}
                </span>
                {user.gradeLevel && (
                  <span className="text-xs text-gray-400">
                    {user.gradeLevel}
                  </span>
                )}
              </div>
            </div>
            <button
              onClick={() => toggleRole(user)}
              className="text-sm text-blue-600 hover:underline"
            >
              Make {user.role === "admin" ? "Student" : "Admin"}
            </button>
          </div>
        ))}
      </div>

      {totalPages > 1 && (
        <div className="mt-6">
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
          />
        </div>
      )}
    </div>
  );
}
