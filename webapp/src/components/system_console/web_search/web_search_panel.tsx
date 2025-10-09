// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import styled from 'styled-components';
import {FormattedMessage, useIntl} from 'react-intl';

import {BooleanItem, ItemList, SelectionItem, SelectionItemOption, TextItem} from '../item';

export type WebSearchGoogleConfig = {
    apiKey: string;
    searchEngineId: string;
    resultLimit: number;
    apiURL: string;
};

export type WebSearchConfig = {
    enabled: boolean;
    provider: string;
    google: WebSearchGoogleConfig;
};

type Props = {
    value: WebSearchConfig;
    onChange: (config: WebSearchConfig) => void;
};

const PanelSubtitle = styled.div`
    color: rgba(var(--center-channel-color-rgb), 0.56);
    font-size: 13px;
    line-height: 18px;
`;

const WebSearchPanel = ({value, onChange}: Props) => {
    const intl = useIntl();

    const handleUpdate = (patch: Partial<WebSearchConfig>) => {
        onChange({...value, ...patch});
    };

    const handleGoogleUpdate = (patch: Partial<WebSearchGoogleConfig>) => {
        handleUpdate({google: {...value.google, ...patch}});
    };

    return (
        <div>
            <PanelSubtitle>
                <FormattedMessage defaultMessage='Configure built-in web search for agents that do not have native provider search capabilities.'/>
            </PanelSubtitle>
            <ItemList>
                <BooleanItem
                    label={intl.formatMessage({defaultMessage: 'Enable Web Search'})}
                    value={value.enabled}
                    onChange={(enabled) => handleUpdate({enabled})}
                    helpText={intl.formatMessage({defaultMessage: 'Allow agents to call Mattermost\'s built-in web search tool. If your LLM already provides native web search support, leave this disabled.'})}
                />
                <SelectionItem
                    label={intl.formatMessage({defaultMessage: 'Provider'})}
                    value={value.provider}
                    onChange={(e) => handleUpdate({provider: e.target.value})}
                    disabled={!value.enabled}
                >
                    <SelectionItemOption value='google'>{'Google Custom Search'}</SelectionItemOption>
                </SelectionItem>
                <TextItem
                    label={intl.formatMessage({defaultMessage: 'Google API Key'})}
                    type='password'
                    value={value.google.apiKey}
                    onChange={(e) => handleGoogleUpdate({apiKey: e.target.value})}
                    disabled={!value.enabled || value.provider !== 'google'}
                />
                <TextItem
                    label={intl.formatMessage({defaultMessage: 'Search Engine ID'})}
                    value={value.google.searchEngineId}
                    onChange={(e) => handleGoogleUpdate({searchEngineId: e.target.value})}
                    disabled={!value.enabled || value.provider !== 'google'}
                />
                <TextItem
                    label={intl.formatMessage({defaultMessage: 'Result Limit'})}
                    type='number'
                    value={value.google.resultLimit.toString()}
                    onChange={(e) => {
                        const parsed = parseInt(e.target.value, 10);
                        handleGoogleUpdate({resultLimit: Number.isNaN(parsed) ? 5 : parsed});
                    }}
                    disabled={!value.enabled || value.provider !== 'google'}
                />
                <TextItem
                    label={intl.formatMessage({defaultMessage: 'API URL (optional)'})}
                    value={value.google.apiURL}
                    onChange={(e) => handleGoogleUpdate({apiURL: e.target.value})}
                    helptext={intl.formatMessage({defaultMessage: 'Override the default Google Custom Search endpoint if necessary.'})}
                    disabled={!value.enabled || value.provider !== 'google'}
                />
            </ItemList>
        </div>
    );
};

export default WebSearchPanel;
