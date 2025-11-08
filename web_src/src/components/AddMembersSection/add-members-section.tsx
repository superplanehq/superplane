"use client";

import { useState } from "react";
import { Text } from "../Text/text";
import { MaterialSymbol } from "../MaterialSymbol/material-symbol";
import { Avatar } from "../Avatar/avatar";
import { MultiCombobox, MultiComboboxLabel } from "../MultiCombobox/multi-combobox";
import { Button } from "../Button/button";

interface User {
  id: string;
  name: string;
  email: string;
  username?: string;
  avatar?: string;
  initials: string;
  type: "member" | "invitation" | "custom";
}

interface AddMembersSectionProps {
  orgUsers: User[];
  onAddMembers: (users: User[]) => Promise<void>;
  isLoading?: boolean;
  disabled?: boolean;
}

export function AddMembersSection({
  orgUsers,
  onAddMembers,
  isLoading = false,
  disabled = false,
}: AddMembersSectionProps) {
  const [selectedUsers, setSelectedUsers] = useState<User[]>([]);

  const isValidEmail = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  };

  const createUserFromEmail = (email: string): User => {
    const name = email.split("@")[0];
    const initials = name.charAt(0).toUpperCase();

    return {
      id: `email_${email}`,
      name: name,
      email: email,
      initials: initials,
      type: "custom",
    };
  };

  const handleAddMembers = async () => {
    if (selectedUsers.length === 0) return;

    try {
      await onAddMembers(selectedUsers);
      setSelectedUsers([]);
    } catch (error) {
      console.error("Error adding members:", error);
    }
  };

  return (
    <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 text-left">
      <div className="flex items-center justify-between mb-4">
        <div>
          <Text className="font-semibold text-zinc-900 dark:text-white mb-1">Add Members</Text>
          <Text className="text-sm text-zinc-600 dark:text-zinc-400">
            Search organization members, pending invites, or invite new users
          </Text>
        </div>
      </div>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleAddMembers();
        }}
        className="flex gap-2"
      >
        <MultiCombobox
          options={orgUsers}
          displayValue={(user) => user.name}
          placeholder="Search users or enter email"
          value={selectedUsers}
          onChange={setSelectedUsers}
          className="flex-1"
          allowCustomValues={true}
          createCustomValue={(query) => {
            const trimmedQuery = query.trim();
            return createUserFromEmail(trimmedQuery);
          }}
          validateValue={(user) => {
            return isValidEmail(user.email);
          }}
          validateInput={(input) => {
            const trimmed = input.trim();
            if (trimmed === "") return false;
            return isValidEmail(trimmed);
          }}
          filter={(user, query) => {
            return (
              user?.username?.toLowerCase().includes(query.toLowerCase()) ||
              user?.name?.toLowerCase().includes(query.toLowerCase()) ||
              user?.email?.toLowerCase().includes(query.toLowerCase()) ||
              false
            );
          }}
          disabled={disabled || isLoading}
        >
          {(user, isSelected) => {
            const isCustomEmailSuggestion = user.type === "custom";

            return (
              <div className="group w-full flex items-center">
                {isCustomEmailSuggestion ? (
                  isSelected ? (
                    <span className="material-symbols-outlined text-xs! text-zinc-600 dark:text-zinc-400 ml-1">
                      mail
                    </span>
                  ) : (
                    <div className="flex items-center justify-center size-8 bg-zinc-100 dark:bg-zinc-800 rounded-full">
                      <MaterialSymbol name="mail" size="md" className="text-zinc-600 dark:text-zinc-400" />
                    </div>
                  )
                ) : (
                  <Avatar src={user.avatar} initials={user.initials} className={isSelected ? "size-4" : "size-8"} />
                )}
                <MultiComboboxLabel className="flex flex-col w-full">
                  {isSelected ? (
                    <span className="font-medium">{isCustomEmailSuggestion ? user.email : user.name || "Unknown"}</span>
                  ) : (
                    <>
                      {isCustomEmailSuggestion ? (
                        <>
                          <span className="font-medium">{user.email}</span>
                          <span className="text-sm text-zinc-600 dark:text-zinc-400 group-hover:text-white">
                            Invite to organization and add to canvas
                          </span>
                        </>
                      ) : (
                        <>
                          <span className="font-medium">{user.name || "Unknown"}</span>
                          <span className="text-sm text-zinc-600 dark:text-zinc-400 group-hover:text-white">
                            {user.email}
                          </span>
                        </>
                      )}
                    </>
                  )}
                </MultiComboboxLabel>
              </div>
            );
          }}
        </MultiCombobox>
        <Button type="submit" disabled={selectedUsers.length === 0 || isLoading || disabled} color="blue">
          <MaterialSymbol name="add" size="sm" />
          {isLoading ? "Adding..." : "Add"}
        </Button>
      </form>
    </div>
  );
}
