// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState, useEffect} from 'react';
import styled from 'styled-components';
import {FormattedMessage} from 'react-intl';

import {doToolCall, getToolPermissions, updateToolPermission} from '@/client';

import {ToolCall, ToolCallStatus} from './llmbot_post/llmbot_post';
import ToolCard from './tool_card';

// Styled components
const ToolCallsContainer = styled.div`
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 12px;
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
    toolCalls: ToolCall[];
}

// Define a type for tool decisions
type ToolDecision = {
    [toolId: string]: boolean | null; // true = approved, false = rejected, null = undecided
};

const ToolApprovalSet: React.FC<ToolApprovalSetProps> = (props) => {
    // Track which tools are currently being processed
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState('');
    const [autoApprovedTools, setAutoApprovedTools] = useState<string[]>([]);

    // Initialize collapsed state - auto-approved tools start collapsed
    const [collapsedTools, setCollapsedTools] = useState<string[]>([]);
    const [toolDecisions, setToolDecisions] = useState<ToolDecision>({});

    // Load permissions from KV store on mount or when postID changes
    useEffect(() => {
        const loadPermissions = async () => {
            try {
                const permissions = await getToolPermissions(props.postID);
                setAutoApprovedTools(permissions);
            } catch (err) {
                // Log error but continue with empty permissions - don't block UI
                setAutoApprovedTools([]);
            }
        };

        loadPermissions();
    }, [props.postID]);

    // Auto-collapse tools that are auto-approved when tool calls arrive or permissions change
    useEffect(() => {
        const autoApprovedToolIds = props.toolCalls.
            filter((tool) => autoApprovedTools.includes(tool.name)).
            map((tool) => tool.id);
        setCollapsedTools(autoApprovedToolIds);
    }, [props.toolCalls, autoApprovedTools]);

    const handleToolDecision = async (toolID: string, approved: boolean, autoApproveTool?: string) => {
        if (isSubmitting) {
            return;
        }

        // Collapse the tool immediately
        setCollapsedTools((prev) => [...prev, toolID]);

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
            await doToolCall(props.postID, approvedToolIDs, autoApproveTool);
        } catch (err) {
            setError('Failed to submit tool decisions');
            setIsSubmitting(false);
        }
    };

    const handleAcceptAll = async (toolID: string, toolName: string) => {
        // Add to local auto-approved state
        setAutoApprovedTools((prev) => [...prev, toolName]);

        // Approve this tool and set auto-approval
        await handleToolDecision(toolID, true, toolName);
    };

    const handlePermissionChange = async (toolName: string, permission: 'ask' | 'auto-approve') => {
        // Update local state optimistically
        if (permission === 'auto-approve') {
            setAutoApprovedTools((prev) => [...prev, toolName]);
        } else {
            setAutoApprovedTools((prev) => prev.filter((t) => t !== toolName));
        }

        // Send update to backend using dedicated permission endpoint
        try {
            await updateToolPermission(props.postID, toolName, permission);
        } catch (err) {
            // Revert optimistic update on error
            if (permission === 'auto-approve') {
                setAutoApprovedTools((prev) => prev.filter((t) => t !== toolName));
            } else {
                setAutoApprovedTools((prev) => [...prev, toolName]);
            }
        }
    };

    const toggleCollapse = (toolID: string) => {
        setCollapsedTools((prev) =>
            (prev.includes(toolID) ? prev.filter((id) => id !== toolID) : [...prev, toolID]),
        );
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

    return (
        <ToolCallsContainer>
            {pendingToolCalls.map((tool) => (
                <ToolCard
                    key={tool.id}
                    tool={tool}
                    isCollapsed={collapsedTools.includes(tool.id)}
                    isProcessing={isSubmitting}
                    onToggleCollapse={() => toggleCollapse(tool.id)}
                    onApprove={() => handleToolDecision(tool.id, true)}
                    onReject={() => handleToolDecision(tool.id, false)}
                    onAcceptAll={() => handleAcceptAll(tool.id, tool.name)}
                    onPermissionChange={(permission) => handlePermissionChange(tool.name, permission)}
                    autoApproved={autoApprovedTools.includes(tool.name)}
                />
            ))}

            {processedToolCalls.map((tool) => (
                <ToolCard
                    key={tool.id}
                    tool={tool}
                    isCollapsed={collapsedTools.includes(tool.id)}
                    isProcessing={false}
                    onToggleCollapse={() => toggleCollapse(tool.id)}
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
