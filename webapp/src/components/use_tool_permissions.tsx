// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {useState, useEffect} from 'react';
import {WebSocketMessage} from '@mattermost/client';

import {getToolPermissions} from '@/client';
import {ToolPermissionWebsocketMessage} from '@/websocket';

interface UseToolPermissionsProps {
    postID: string;
    rootPostID: string;
    websocketRegister?: (rootPostID: string, listenerID: string, handler: (msg: WebSocketMessage<any>) => void) => void;
    websocketUnregister?: (rootPostID: string, listenerID: string) => void;
}

interface UseToolPermissionsReturn {
    autoApprovedTools: string[];
    permissionsLoading: boolean;
}

/**
 * Custom hook to manage tool permissions for a conversation.
 * Handles loading permissions from the server and listening for real-time updates via WebSocket.
 */
export function useToolPermissions({
    postID,
    rootPostID,
    websocketRegister,
    websocketUnregister,
}: UseToolPermissionsProps): UseToolPermissionsReturn {
    const [autoApprovedTools, setAutoApprovedTools] = useState<string[]>([]);
    const [permissionsLoading, setPermissionsLoading] = useState(true);

    // Load permissions from KV store on mount or when postID changes
    useEffect(() => {
        const loadPermissions = async () => {
            setPermissionsLoading(true);
            try {
                const permissions = await getToolPermissions(postID);
                setAutoApprovedTools(permissions);
            } catch (err) {
                // Log error but continue with empty permissions - don't block UI
                setAutoApprovedTools([]);
            } finally {
                setPermissionsLoading(false);
            }
        };

        loadPermissions();
    }, [postID]);

    // Register WebSocket listener for tool permission updates
    useEffect(() => {
        if (websocketRegister && websocketUnregister) {
            const listenerID = `tool-permissions-${postID}`;

            const handleToolPermissionUpdate = (msg: WebSocketMessage<ToolPermissionWebsocketMessage>) => {
                const {tool_name, permission} = msg.data;

                // Update local state based on the permission change
                if (permission === 'auto-approve') {
                    setAutoApprovedTools((prev) => {
                        if (!prev.includes(tool_name)) {
                            return [...prev, tool_name];
                        }
                        return prev;
                    });
                } else {
                    setAutoApprovedTools((prev) => prev.filter((t) => t !== tool_name));
                }
            };

            websocketRegister(rootPostID, listenerID, handleToolPermissionUpdate);

            return () => {
                if (websocketUnregister) {
                    websocketUnregister(rootPostID, listenerID);
                }
            };
        }

        return () => {/* no cleanup */};
    }, [rootPostID, postID, websocketRegister, websocketUnregister]);

    return {
        autoApprovedTools,
        permissionsLoading,
    };
}
