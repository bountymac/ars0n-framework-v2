// Define the auto scan steps
const AUTO_SCAN_STEPS = {
  IDLE: 'idle', // 0
  AMASS: 'amass', // 1
  SUBLIST3R: 'sublist3r', // 2
  ASSETFINDER: 'assetfinder', // 3
  GAU: 'gau', // 4
  CTL: 'ctl', // 5
  SUBFINDER: 'subfinder', // 6
  CONSOLIDATE: 'consolidate', // 7
  HTTPX: 'httpx', // 8
  SHUFFLEDNS: 'shuffledns', // 9
  SHUFFLEDNS_CEWL: 'shuffledns_cewl', // 10
  CONSOLIDATE_ROUND2: 'consolidate_round2', // 10.5
  HTTPX_ROUND2: 'httpx_round2', // 10.75
  GOSPIDER: 'gospider', // 11
  SUBDOMAINIZER: 'subdomainizer', // 12
  CONSOLIDATE_ROUND3: 'consolidate_round3', // 12.5
  HTTPX_ROUND3: 'httpx_round3', // 12.75
  NUCLEI_SCREENSHOT: 'nuclei-screenshot', // 13
  METADATA: 'metadata', // 14
  COMPLETED: 'completed' // 15
};

// Debug utility function
const debugTrace = (message) => {
  console.log(`[Auto-Scan Debug] ${message}`);
};

// Utility function to update the auto scan state on the server
const updateAutoScanState = async (targetId, currentStep, isPaused = false, isCancelled = false, config = null) => {
  try {
    // Skip updating disabled steps in the server state
    if (config && currentStep !== AUTO_SCAN_STEPS.IDLE && currentStep !== AUTO_SCAN_STEPS.COMPLETED) {
      const stepConfigMapping = {
        [AUTO_SCAN_STEPS.AMASS]: 'amass',
        [AUTO_SCAN_STEPS.SUBLIST3R]: 'sublist3r',
        [AUTO_SCAN_STEPS.ASSETFINDER]: 'assetfinder',
        [AUTO_SCAN_STEPS.GAU]: 'gau',
        [AUTO_SCAN_STEPS.CTL]: 'ctl',
        [AUTO_SCAN_STEPS.SUBFINDER]: 'subfinder',
        [AUTO_SCAN_STEPS.CONSOLIDATE]: 'consolidate_httpx_round1',
        [AUTO_SCAN_STEPS.HTTPX]: 'consolidate_httpx_round1',
        [AUTO_SCAN_STEPS.SHUFFLEDNS]: 'shuffledns',
        [AUTO_SCAN_STEPS.SHUFFLEDNS_CEWL]: 'cewl',
        [AUTO_SCAN_STEPS.CONSOLIDATE_ROUND2]: 'consolidate_httpx_round2',
        [AUTO_SCAN_STEPS.HTTPX_ROUND2]: 'consolidate_httpx_round2',
        [AUTO_SCAN_STEPS.GOSPIDER]: 'gospider',
        [AUTO_SCAN_STEPS.SUBDOMAINIZER]: 'subdomainizer',
        [AUTO_SCAN_STEPS.CONSOLIDATE_ROUND3]: 'consolidate_httpx_round3',
        [AUTO_SCAN_STEPS.HTTPX_ROUND3]: 'consolidate_httpx_round3',
        [AUTO_SCAN_STEPS.NUCLEI_SCREENSHOT]: 'nuclei_screenshot',
        [AUTO_SCAN_STEPS.METADATA]: 'metadata'
      };
      
      const configKey = stepConfigMapping[currentStep];
      if (configKey && config[configKey] === false) {
        // This step is disabled in config, skip updating the state
        debugTrace(`Skipping update of disabled step "${currentStep}" in server state`);
        return true;
      }
    }

    const response = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${targetId}`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ 
          current_step: currentStep,
          is_paused: isPaused,
          is_cancelled: isCancelled
        }),
      }
    );
    
    if (!response.ok) {
      throw new Error(`Failed to update auto scan state: ${response.statusText}`);
    }
    
    debugTrace(`API updated: current_step=${currentStep}, is_paused=${isPaused}, is_cancelled=${isCancelled}`);
    return true;
  } catch (error) {
    debugTrace(`Error updating auto scan state: ${error.message}`);
    return false;
  }
};

// Helper function to wait for a scan to complete
const waitForScanCompletion = async (scanType, targetId, setIsScanning, setMostRecentScanStatus, setMostRecentScan = null) => {
  debugTrace(`waitForScanCompletion started for ${scanType}`);

  return new Promise((resolve) => {
    let attempts = 0;
    const checkStatus = async () => {
      attempts++;
      debugTrace(`Checking ${scanType} scan status - attempt #${attempts}`);
      try {
        const url = `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${targetId}/scans/${scanType}`;
        debugTrace(`Fetching scan status from: ${url}`);
        const response = await fetch(url);
        if (!response.ok) {
          debugTrace(`Failed to fetch ${scanType} scan status: ${response.status} ${response.statusText}`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const data = await response.json();
        const scans = Array.isArray(data) ? data : (data.scans || []);
        debugTrace(`Retrieved ${scans.length} ${scanType} scans`);
        if (!scans || !Array.isArray(scans) || scans.length === 0) {
          debugTrace(`No ${scanType} scans found, will check again`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const mostRecentScan = scans.reduce((latest, scan) => {
          const scanDate = new Date(scan.created_at);
          return scanDate > new Date(latest.created_at) ? scan : latest;
        }, scans[0]);
        debugTrace(`Most recent ${scanType} scan status: ${mostRecentScan.status}, ID: ${mostRecentScan.id || 'unknown'}`);
        setMostRecentScanStatus(mostRecentScan.status);
        if (setMostRecentScan) {
          setMostRecentScan(mostRecentScan);
          debugTrace(`Updated UI with most recent ${scanType} scan data`);
        }
        if (mostRecentScan.status === 'completed' || 
            mostRecentScan.status === 'success' || 
            mostRecentScan.status === 'failed' || 
            mostRecentScan.status === 'error') {
          debugTrace(`${scanType} scan finished with status: ${mostRecentScan.status}`);
          setIsScanning(false);
          resolve(mostRecentScan);
        } else if (mostRecentScan.status === 'processing') {
          debugTrace(`${scanType} scan is still processing large results, checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        } else {
          debugTrace(`${scanType} scan still pending (status: ${mostRecentScan.status}), checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        }
      } catch (error) {
        debugTrace(`Error checking ${scanType} scan status: ${error.message}\n${error.stack}`);
        setTimeout(checkStatus, 5000);
      }
    };
    checkStatus();
  });
};

// Custom function to wait for both CeWL and ShuffleDNS Custom scans
const waitForCeWLAndShuffleDNSCustom = async (
  activeTarget,
  setIsCeWLScanning,
  setMostRecentCeWLScanStatus,
  setMostRecentCeWLScan,
  setMostRecentShuffleDNSCustomScanStatus,
  setMostRecentShuffleDNSCustomScan
) => {
  debugTrace(`Starting waitForCeWLAndShuffleDNSCustom for target ${activeTarget.id}`);
  await waitForScanCompletion(
    'cewl',
    activeTarget.id,
    setIsCeWLScanning,
    setMostRecentCeWLScanStatus,
    setMostRecentCeWLScan
  );
  debugTrace(`CeWL scan completed, now waiting for ShuffleDNS custom scan`);
  return new Promise((resolve) => {
    let attempts = 0;
    const checkStatus = async () => {
      attempts++;
      debugTrace(`Checking ShuffleDNS custom scan status - attempt #${attempts}`);
      try {
        const url = `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/scope-targets/${activeTarget.id}/shufflednscustom-scans`;
        debugTrace(`Fetching ShuffleDNS custom scan status from: ${url}`);
        const response = await fetch(url);
        if (!response.ok) {
          debugTrace(`Failed to fetch ShuffleDNS custom scan status: ${response.status} ${response.statusText}`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const scans = await response.json();
        debugTrace(`Retrieved ${scans?.length || 0} ShuffleDNS custom scans`);
        if (!scans || !Array.isArray(scans) || scans.length === 0) {
          debugTrace(`No ShuffleDNS custom scans found, will check again`);
          setTimeout(checkStatus, 5000);
          return;
        }
        const mostRecentScan = scans[0];
        debugTrace(`Most recent ShuffleDNS custom scan status: ${mostRecentScan.status}, ID: ${mostRecentScan.id || 'unknown'}`);
        setMostRecentShuffleDNSCustomScanStatus(mostRecentScan.status);
        setMostRecentShuffleDNSCustomScan(mostRecentScan);
        if (mostRecentScan.status === 'completed' || 
            mostRecentScan.status === 'success' || 
            mostRecentScan.status === 'failed' || 
            mostRecentScan.status === 'error') {
          debugTrace(`ShuffleDNS custom scan finished with status: ${mostRecentScan.status}`);
          resolve(mostRecentScan);
        } else if (mostRecentScan.status === 'processing') {
          debugTrace(`ShuffleDNS custom scan is still processing large results, checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        } else {
          debugTrace(`ShuffleDNS custom scan still pending (status: ${mostRecentScan.status}), checking again in 5 seconds`);
          setTimeout(checkStatus, 5000);
        }
      } catch (error) {
        debugTrace(`Error checking ShuffleDNS custom scan status: ${error.message}\n${error.stack}`);
        setTimeout(checkStatus, 5000);
      }
    };
    checkStatus();
  });
};

// Function to resume auto scan from a specific step
const resumeAutoScan = async (
  fromStep,
  activeTarget,
  getAutoScanSteps,
  setIsAutoScanning,
  setAutoScanCurrentStep
) => {
  try {
    setIsAutoScanning(true);
    
    // Get the config for this auto scan
    const configResponse = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-config`
    );
    
    if (!configResponse.ok) {
      throw new Error("Failed to fetch auto scan config");
    }
    
    const config = await configResponse.json();
    debugTrace("Auto scan config retrieved for resume");
    
    // Get the current state for checking limits
    let consolidatedSubdomains = [];
    let mostRecentHttpxScan = null;
    
    try {
      // Fetch current consolidated subdomains
      const subdomainsResponse = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/consolidated-subdomains/${activeTarget.id}`
      );
      if (subdomainsResponse.ok) {
        const data = await subdomainsResponse.json();
        consolidatedSubdomains = data.subdomains || [];
      }
      
      // Fetch current HTTPX scan
      const httpxResponse = await fetch(
        `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/scopetarget/${activeTarget.id}/scans/httpx`
      );
      if (httpxResponse.ok) {
        const scans = await httpxResponse.json();
        if (Array.isArray(scans) && scans.length > 0) {
          mostRecentHttpxScan = scans.reduce((latest, scan) => {
            const scanDate = new Date(scan.created_at);
            return scanDate > new Date(latest.created_at) ? scan : latest;
          }, scans[0]);
        }
      }
    } catch (error) {
      debugTrace("Error fetching current state for limit checks: " + error.message);
    }
    
    let startFromIndex = 0;
    const steps = getAutoScanSteps(activeTarget, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, config, undefined, 
      mostRecentHttpxScan, consolidatedSubdomains);
      
    for (let i = 0; i < steps.length; i++) {
      if (steps[i].name === fromStep) {
        startFromIndex = i;
        break;
      }
    }
    
    // Execute steps from the determined starting point
    for (let i = startFromIndex; i < steps.length; i++) {
      try {
        // Check if cancelled before starting the step
        const stateResponse = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
        );
        if (stateResponse.ok) {
          const state = await stateResponse.json();
          // If cancelled, exit the loop
          if (state.is_cancelled) {
            debugTrace("Auto scan was cancelled, ending early");
            await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED, false, false, config);
            break;
          }
        }
        
        // Update current step - skip if disabled
        const stepConfigMapping = {
          [AUTO_SCAN_STEPS.AMASS]: 'amass',
          [AUTO_SCAN_STEPS.SUBLIST3R]: 'sublist3r',
          [AUTO_SCAN_STEPS.ASSETFINDER]: 'assetfinder',
          [AUTO_SCAN_STEPS.GAU]: 'gau',
          [AUTO_SCAN_STEPS.CTL]: 'ctl',
          [AUTO_SCAN_STEPS.SUBFINDER]: 'subfinder',
          [AUTO_SCAN_STEPS.CONSOLIDATE]: 'consolidate_httpx_round1',
          [AUTO_SCAN_STEPS.HTTPX]: 'consolidate_httpx_round1',
          [AUTO_SCAN_STEPS.SHUFFLEDNS]: 'shuffledns',
          [AUTO_SCAN_STEPS.SHUFFLEDNS_CEWL]: 'cewl',
          [AUTO_SCAN_STEPS.CONSOLIDATE_ROUND2]: 'consolidate_httpx_round2',
          [AUTO_SCAN_STEPS.HTTPX_ROUND2]: 'consolidate_httpx_round2',
          [AUTO_SCAN_STEPS.GOSPIDER]: 'gospider',
          [AUTO_SCAN_STEPS.SUBDOMAINIZER]: 'subdomainizer',
          [AUTO_SCAN_STEPS.CONSOLIDATE_ROUND3]: 'consolidate_httpx_round3',
          [AUTO_SCAN_STEPS.HTTPX_ROUND3]: 'consolidate_httpx_round3',
          [AUTO_SCAN_STEPS.NUCLEI_SCREENSHOT]: 'nuclei_screenshot',
          [AUTO_SCAN_STEPS.METADATA]: 'metadata'
        };
        
        const stepName = steps[i].name;
        const configKey = stepConfigMapping[stepName];
        
        // Only update UI state if the step is enabled or it's a system step
        if (!configKey || stepName === AUTO_SCAN_STEPS.IDLE || stepName === AUTO_SCAN_STEPS.COMPLETED || config[configKey] !== false) {
          setAutoScanCurrentStep(stepName);
          await updateAutoScanState(activeTarget.id, stepName, false, false, config);
        }
        
        // Run the current step
        await steps[i].action();
        
        // Check if paused or cancelled after step completes
        const pauseResponse = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
        );
        if (pauseResponse.ok) {
          const state = await pauseResponse.json();
          // If cancelled, exit the loop
          if (state.is_cancelled) {
            debugTrace("Auto scan was cancelled after step completion, ending early");
            await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED, false, false, config);
            break;
          }
          
          // If paused, wait until unpaused
          if (state.is_paused) {
            debugTrace(`Auto scan paused after step ${steps[i].name}`);
            // Mark the current step as paused
            setAutoScanCurrentStep(`${steps[i].name}`);
            
            // Wait until unpaused
            let isPaused = true;
            while (isPaused) {
              await new Promise(resolve => setTimeout(resolve, 2000));
              const checkResponse = await fetch(
                `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
              );
              if (checkResponse.ok) {
                const checkState = await checkResponse.json();
                // If cancelled while paused, exit the loop
                if (checkState.is_cancelled) {
                  debugTrace("Auto scan was cancelled while paused");
                  await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED, false, false, config);
                  return;
                }
                // If unpaused, continue
                if (!checkState.is_paused) {
                  debugTrace("Auto scan resumed");
                  isPaused = false;
                }
              }
            }
          }
        }
      } catch (error) {
        debugTrace(`Error in step ${i+1}/${steps.length}: ${error.message}`);
      }
      
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
    
  } catch (error) {
    debugTrace(`Error resuming Auto Scan: ${error.message}`);
  } finally {
    setIsAutoScanning(false);
    setAutoScanCurrentStep(AUTO_SCAN_STEPS.COMPLETED);
    
    // Update the API with completed status - don't need to check if this is disabled
    await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED);
  }
};

// Function to start a new auto scan
const startAutoScan = async (
  activeTarget,
  setIsAutoScanning,
  setAutoScanCurrentStep,
  setAutoScanTargetId,
  getAutoScanSteps,
  consolidatedSubdomains,
  mostRecentHttpxScan,
  autoScanSessionId
) => {
  if (!activeTarget || !activeTarget.id) {
    console.log("No active target selected.");
    return;
  }

  setIsAutoScanning(true);
  setAutoScanCurrentStep(AUTO_SCAN_STEPS.IDLE);
  setAutoScanTargetId(activeTarget.id);
  
  await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.IDLE);
  
  try {
    // Get the config for this auto scan
    const configResponse = await fetch(
      `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-config`
    );
    
    if (!configResponse.ok) {
      throw new Error("Failed to fetch auto scan config");
    }
    
    const config = await configResponse.json();
    debugTrace("Auto scan config retrieved");
    
    const steps = getAutoScanSteps(undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, undefined, 
      undefined, undefined, undefined, undefined, undefined, undefined, undefined, config, autoScanSessionId, 
      mostRecentHttpxScan, consolidatedSubdomains);
    for (let i = 0; i < steps.length; i++) {
      try {
        // Update current step - skip UI update if step is disabled
        const stepConfigMapping = {
          [AUTO_SCAN_STEPS.AMASS]: 'amass',
          [AUTO_SCAN_STEPS.SUBLIST3R]: 'sublist3r',
          [AUTO_SCAN_STEPS.ASSETFINDER]: 'assetfinder',
          [AUTO_SCAN_STEPS.GAU]: 'gau',
          [AUTO_SCAN_STEPS.CTL]: 'ctl',
          [AUTO_SCAN_STEPS.SUBFINDER]: 'subfinder',
          [AUTO_SCAN_STEPS.CONSOLIDATE]: 'consolidate_httpx_round1',
          [AUTO_SCAN_STEPS.HTTPX]: 'consolidate_httpx_round1',
          [AUTO_SCAN_STEPS.SHUFFLEDNS]: 'shuffledns',
          [AUTO_SCAN_STEPS.SHUFFLEDNS_CEWL]: 'cewl',
          [AUTO_SCAN_STEPS.CONSOLIDATE_ROUND2]: 'consolidate_httpx_round2',
          [AUTO_SCAN_STEPS.HTTPX_ROUND2]: 'consolidate_httpx_round2',
          [AUTO_SCAN_STEPS.GOSPIDER]: 'gospider',
          [AUTO_SCAN_STEPS.SUBDOMAINIZER]: 'subdomainizer',
          [AUTO_SCAN_STEPS.CONSOLIDATE_ROUND3]: 'consolidate_httpx_round3',
          [AUTO_SCAN_STEPS.HTTPX_ROUND3]: 'consolidate_httpx_round3',
          [AUTO_SCAN_STEPS.NUCLEI_SCREENSHOT]: 'nuclei_screenshot',
          [AUTO_SCAN_STEPS.METADATA]: 'metadata'
        };
        
        const stepName = steps[i].name;
        const configKey = stepConfigMapping[stepName];
        
        // Only update UI state if the step is enabled or it's a system step
        if (!configKey || stepName === AUTO_SCAN_STEPS.IDLE || stepName === AUTO_SCAN_STEPS.COMPLETED || config[configKey] !== false) {
          setAutoScanCurrentStep(stepName);
          await updateAutoScanState(activeTarget.id, stepName, false, false, config);
        }
        
        // Check if cancelled before starting the step
        const stateResponse = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
        );
        if (stateResponse.ok) {
          const state = await stateResponse.json();
          // If cancelled, exit the loop
          if (state.is_cancelled) {
            debugTrace("Auto scan was cancelled, ending early");
            await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED, false, false, config);
            break;
          }
        }
        
        // Run the step
        await steps[i].action();
        
        // Check if paused or cancelled after step completes
        const pauseResponse = await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
        );
        if (pauseResponse.ok) {
          const state = await pauseResponse.json();
          // If cancelled, exit the loop
          if (state.is_cancelled) {
            debugTrace("Auto scan was cancelled after step completion, ending early");
            await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED, false, false, config);
            break;
          }
          
          // If paused, wait until unpaused
          if (state.is_paused) {
            debugTrace(`Auto scan paused after step ${steps[i].name}`);
            
            // Wait until unpaused
            let isPaused = true;
            while (isPaused) {
              await new Promise(resolve => setTimeout(resolve, 2000));
              const checkResponse = await fetch(
                `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan-state/${activeTarget.id}`
              );
              if (checkResponse.ok) {
                const checkState = await checkResponse.json();
                // If cancelled while paused, exit the loop
                if (checkState.is_cancelled) {
                  debugTrace("Auto scan was cancelled while paused");
                  await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED, false, false, config);
                  return;
                }
                // If unpaused, continue
                if (!checkState.is_paused) {
                  debugTrace("Auto scan resumed");
                  isPaused = false;
                }
              }
            }
          }
        }
      } catch (error) {
        debugTrace(`Error in step ${steps[i].name}: ${error.message}`);
      }
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
    debugTrace("All auto scan steps completed");
  } catch (error) {
    debugTrace(`ERROR during Auto Scan: ${error.message}`);
  } finally {
    debugTrace("Auto Scan session finalizing - setting state to COMPLETED");
    setIsAutoScanning(false);
    setAutoScanCurrentStep(AUTO_SCAN_STEPS.COMPLETED);
    await updateAutoScanState(activeTarget.id, AUTO_SCAN_STEPS.COMPLETED);
    debugTrace("Auto Scan session ended");
    if (autoScanSessionId) {
      let finalConsolidatedSubdomains = Array.isArray(consolidatedSubdomains) ? consolidatedSubdomains.length : 0;
      let finalLiveWebServers = 0;
      if (mostRecentHttpxScan && mostRecentHttpxScan.result && typeof mostRecentHttpxScan.result.String === 'string') {
        finalLiveWebServers = mostRecentHttpxScan.result.String.split('\n').filter(line => line.trim()).length;
      }
      try {
        await fetch(
          `${process.env.REACT_APP_SERVER_PROTOCOL}://${process.env.REACT_APP_SERVER_IP}:${process.env.REACT_APP_SERVER_PORT}/api/auto-scan/session/${autoScanSessionId}/final-stats`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              final_consolidated_subdomains: finalConsolidatedSubdomains,
              final_live_web_servers: finalLiveWebServers
            })
          }
        );
      } catch (err) {
        debugTrace('Failed to update final stats for auto scan session: ' + err.message);
      }
    }
  }
};

// Helper to check and resume auto scan
const checkAndResumeAutoScan = (
  storedStep,
  storedTargetId,
  scopeTargets,
  activeTarget,
  setIsAutoScanning,
  setAutoScanCurrentStep,
  setAutoScanTargetId,
  resumeAutoScanFn
) => {
  if (storedStep && storedStep !== AUTO_SCAN_STEPS.IDLE && storedStep !== AUTO_SCAN_STEPS.COMPLETED && storedTargetId) {
    console.log(`Detected in-progress Auto Scan (step: ${storedStep}). Attempting to resume...`);
    
    // Find the target with the matching ID
    const matchingTarget = scopeTargets.find(target => target.id === storedTargetId);
    
    if (matchingTarget && matchingTarget.id === activeTarget?.id) {
      setIsAutoScanning(true);
      resumeAutoScanFn(storedStep);
    } else {
      // Reset the auto scan state on the server
      updateAutoScanState(storedTargetId, AUTO_SCAN_STEPS.IDLE);
      
      setAutoScanCurrentStep(AUTO_SCAN_STEPS.IDLE);
      setAutoScanTargetId(null);
    }
  }
};

export {
  waitForScanCompletion,
  AUTO_SCAN_STEPS,
  debugTrace,
  resumeAutoScan,
  startAutoScan,
  checkAndResumeAutoScan,
  waitForCeWLAndShuffleDNSCustom,
  updateAutoScanState
}; 