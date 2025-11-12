// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import styled from 'styled-components';
import {FormattedMessage} from 'react-intl';
import {WebSocketMessage} from '@mattermost/client';

import {doToolCall, updateToolPermission} from '@/client';

import {ToolCall, ToolCallStatus} from './llmbot_post/llmbot_post';
import ToolCard from './tool_card';
import {useToolPermissions} from './use_tool_permissions';

// Styled components
const ToolCallsContainer = styled.div`
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin: 16px 0;
`;

const StatusBar = styled.div`
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    margin-top: 8px;
    background: rgba(var(--center-channel-color-rgb), 0.04);
    border-radius: 4px;
    font-size: 12px;
`;

// Tool call interfaces
interface ToolApprovalSetProps {
    postID: string;
    rootPostID: string;
    toolCalls: ToolCall[];
    websocketRegister?: (rootPostID: string, listenerID: string, handler: (msg: WebSocketMessage<any>) => void) => void;
    websocketUnregister?: (rootPostID: string, listenerID: string) => void;
}

// Define a type for tool decisions
type ToolDecision = {
    [toolId: string]: boolean | null; // true = approved, false = rejected, null = undecided
};

const ToolApprovalSet: React.FC<ToolApprovalSetProps> = (props) => {
    // Track which tools are currently being processed
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState('');

    // Use custom hook to manage permissions and WebSocket updates
    const {autoApprovedTools, permissionsLoading} = useToolPermissions({
        postID: props.postID,
        rootPostID: props.rootPostID,
        websocketRegister: props.websocketRegister,
        websocketUnregister: props.websocketUnregister,
    });

    // Track user manual overrides of the default collapse state
    const [userExpandedTools, setUserExpandedTools] = useState<string[]>([]); // User clicked to expand a normally-collapsed tool
    const [userCollapsedTools, setUserCollapsedTools] = useState<string[]>([]); // User clicked to collapse a normally-expanded tool
    const [toolDecisions, setToolDecisions] = useState<ToolDecision>({});

    const handleToolDecision = async (toolID: string, approved: boolean) => {
        if (isSubmitting) {
            return;
        }

        const updatedDecisions = {
            ...toolDecisions,
            [toolID]: approved,
        };
        setToolDecisions(updatedDecisions);

        const hasUndecided = props.toolCalls.some((tool) => {
            return !Object.hasOwn(updatedDecisions, tool.id) || updatedDecisions[tool.id] === null;
        });

        if (hasUndecided) {
            // If there are still undecided tools, do not submit yet
            return;
        }

        // Submit when all tools are decided

        const approvedToolIDs = Object.entries(updatedDecisions).
            filter(([, isApproved]) => {
                return isApproved;
            }).
            map(([id]) => id);

        setIsSubmitting(true);
        try {
            await doToolCall(props.postID, approvedToolIDs);
        } catch (err) {
            setError('Failed to submit tool decisions');
            setIsSubmitting(false);
        }
    };

    const handleAcceptAll = async (toolID: string, toolName: string) => {
        if (isSubmitting) {
            return;
        }

        // Update permission via dedicated endpoint
        // The WebSocket will update the autoApprovedTools state via the custom hook
        try {
            await updateToolPermission(props.postID, toolName, 'auto-approve');
        } catch (err) {
            setError('Failed to update tool permission');
            return;
        }

        // Find ALL pending tools with the same name and approve them all
        const matchingToolIDs = props.toolCalls.
            filter((tool) => tool.name === toolName && tool.status === ToolCallStatus.Pending).
            map((tool) => tool.id);

        // Mark all matching tools as approved
        const updatedDecisions = {
            ...toolDecisions,
        };
        matchingToolIDs.forEach((id) => {
            updatedDecisions[id] = true;
        });
        setToolDecisions(updatedDecisions);

        // Check if there are still undecided tools
        const hasUndecided = props.toolCalls.some((tool) => {
            return !Object.hasOwn(updatedDecisions, tool.id) || updatedDecisions[tool.id] === null;
        });

        if (hasUndecided) {
            // If there are still undecided tools, do not submit yet
            return;
        }

        // Submit when all tools are decided
        const approvedToolIDs = Object.entries(updatedDecisions).
            filter(([, isApproved]) => {
                return isApproved;
            }).
            map(([id]) => id);

        setIsSubmitting(true);
        try {
            await doToolCall(props.postID, approvedToolIDs);
        } catch (err) {
            setError('Failed to submit tool decisions');
            setIsSubmitting(false);
        }
    };

    const handlePermissionChange = async (toolName: string, permission: 'ask' | 'auto-approve') => {
        // Send update to backend using dedicated permission endpoint
        // The WebSocket will update the autoApprovedTools state via the custom hook
        try {
            await updateToolPermission(props.postID, toolName, permission);
        } catch (err) {
            setError('Failed to update tool permission');
        }
    };

    const toggleCollapse = (toolID: string, toolName: string, isPending: boolean) => {
        // Determine what the default state should be
        const shouldBeExpandedByDefault = isPending && !autoApprovedTools.includes(toolName);

        if (shouldBeExpandedByDefault) {
            // Default is expanded, so user is clicking to collapse
            setUserCollapsedTools((prev) =>
                (prev.includes(toolID) ? prev.filter((id) => id !== toolID) : [...prev, toolID]),
            );

            // Remove from expanded list if it was there
            setUserExpandedTools((prev) => prev.filter((id) => id !== toolID));
        } else {
            // Default is collapsed, so user is clicking to expand
            setUserExpandedTools((prev) =>
                (prev.includes(toolID) ? prev.filter((id) => id !== toolID) : [...prev, toolID]),
            );

            // Remove from collapsed list if it was there
            setUserCollapsedTools((prev) => prev.filter((id) => id !== toolID));
        }
    };

    if (props.toolCalls.length === 0) {
        return null;
    }

    if (error) {
        return <div className='error'>{error}</div>;
    }

    // Get pending tool calls
    const pendingToolCalls = props.toolCalls.filter((call) => call.status === ToolCallStatus.Pending);

    // Get processed tool calls
    const processedToolCalls = props.toolCalls.filter((call) => call.status !== ToolCallStatus.Pending);

    // Calculate how many tools are left to decide on
    const undecidedCount = Object.values(toolDecisions).filter((decision) => decision === null).length;

    // Helper to compute if a tool should be collapsed
    const isToolCollapsed = (tool: ToolCall) => {
        // Default state: collapsed for everything EXCEPT pending non-auto-approved tools
        const defaultExpanded = tool.status === ToolCallStatus.Pending && !autoApprovedTools.includes(tool.name);

        // Check for user overrides
        const userWantsExpanded = userExpandedTools.includes(tool.id);
        const userWantsCollapsed = userCollapsedTools.includes(tool.id);

        // User overrides take precedence
        if (userWantsExpanded) {
            return false; // Not collapsed (expanded)
        }
        if (userWantsCollapsed) {
            return true; // Collapsed
        }

        // Otherwise use default
        return !defaultExpanded;
    };

    return (
        <ToolCallsContainer>
            {pendingToolCalls.map((tool) => (
                <ToolCard
                    key={tool.id}
                    tool={tool}
                    isCollapsed={isToolCollapsed(tool)}
                    isProcessing={isSubmitting}
                    onToggleCollapse={() => toggleCollapse(tool.id, tool.name, true)}
                    onApprove={() => handleToolDecision(tool.id, true)}
                    onReject={() => handleToolDecision(tool.id, false)}
                    decision={toolDecisions[tool.id]}
                    onAcceptAll={() => handleAcceptAll(tool.id, tool.name)}
                    onPermissionChange={(permission) => handlePermissionChange(tool.name, permission)}
                    autoApproved={autoApprovedTools.includes(tool.name)}
                    permissionsLoading={permissionsLoading}
                />
            ))}

            {processedToolCalls.map((tool) => (
                <ToolCard
                    key={tool.id}
                    tool={tool}
                    isCollapsed={isToolCollapsed(tool)}
                    isProcessing={false}
                    onToggleCollapse={() => toggleCollapse(tool.id, tool.name, false)}
                    onPermissionChange={(permission) => handlePermissionChange(tool.name, permission)}
                    autoApproved={autoApprovedTools.includes(tool.name)}
                />
            ))}

            {/* Only show status bar for multiple pending tools */}
            {pendingToolCalls.length > 1 && isSubmitting && (
                <StatusBar>
                    <div>
                        <FormattedMessage
                            id='ai.tool_call.submitting'
                            defaultMessage='Submitting...'
                        />
                    </div>
                </StatusBar>
            )}

            {/* Only show status counter for multiple pending tools that haven't been submitted yet */}
            {pendingToolCalls.length > 1 && undecidedCount > 0 && !isSubmitting && (
                <StatusBar>
                    <div>
                        <FormattedMessage
                            id='ai.tool_call.pending_decisions'
                            defaultMessage='{count, plural, =0 {All tools decided} one {# tool needs a decision} other {# tools need decisions}}'
                            values={{count: undecidedCount}}
                        />
                    </div>
                </StatusBar>
            )}
        </ToolCallsContainer>
    );
};

export default ToolApprovalSet;
