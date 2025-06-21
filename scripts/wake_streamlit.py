#!/usr/bin/env python3
"""
Streamlit App Wake-Up Script using Playwright

This script uses Playwright to visit Streamlit apps and click the wake-up button
if the app is sleeping. It's designed to be called from the Go cron service.

Usage:
    python wake_streamlit.py <url1> <url2> <url3> ...

Example:
    python wake_streamlit.py https://app1.streamlit.app/ https://app2.streamlit.app/
"""

import sys
import time
import argparse
from typing import List
from playwright.sync_api import sync_playwright, Browser, Page, TimeoutError as PlaywrightTimeoutError

class StreamlitWakeUp:
    def __init__(self, headless: bool = True, timeout: int = 30000):
        """
        Initialize the Streamlit wake-up service.
        
        Args:
            headless: Whether to run browser in headless mode
            timeout: Timeout in milliseconds for page operations
        """
        self.headless = headless
        self.timeout = timeout
        self.browser = None
        
    def __enter__(self):
        """Context manager entry - start Playwright and browser"""
        self.playwright = sync_playwright().start()
        self.browser = self.playwright.chromium.launch(
            headless=self.headless,
            args=[
                '--no-sandbox',
                '--disable-dev-shm-usage',
                '--disable-extensions',
                '--disable-gpu',
                '--disable-background-timer-throttling',
                '--disable-backgrounding-occluded-windows',
                '--disable-renderer-backgrounding'
            ]
        )
        return self
        
    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit - cleanup browser and Playwright"""
        if self.browser:
            self.browser.close()
        if hasattr(self, 'playwright'):
            self.playwright.stop()
    
    def wake_up_app(self, url: str) -> dict:
        """
        Wake up a single Streamlit app.
        
        Args:
            url: The URL of the Streamlit app
            
        Returns:
            dict: Result information about the wake-up attempt
        """
        result = {
            'url': url,
            'success': False,
            'status': 'unknown',
            'message': '',
            'response_time': 0
        }
        
        page = None
        start_time = time.time()
        
        try:
            print(f"🔍 Checking {url}...")
            
            # Create new page
            page = self.browser.new_page()
            
            # Set a reasonable timeout
            page.set_default_timeout(self.timeout)
            
            # Navigate to the app
            response = page.goto(url, wait_until='networkidle')
            result['response_time'] = time.time() - start_time
            
            if response and response.status:
                print(f"   📡 HTTP Status: {response.status}")
            
            # Wait a moment for the page to fully load
            page.wait_for_timeout(3000)
            
            # Look for various wake-up button patterns
            wake_up_selectors = [
                "button:has-text('Yes, get this app back up!')",
                "button:has-text('Wake up')",
                "button:has-text('Start app')",
                "button:has-text('Rerun')",
                "button[data-testid='stButton'] >> text='Yes, get this app back up!'",
                ".stButton button:has-text('Yes, get this app back up!')",
                # More generic selectors
                "button:text-matches('.*back up.*', 'i')",
                "button:text-matches('.*wake.*', 'i')"
            ]
            
            button_found = False
            
            for selector in wake_up_selectors:
                try:
                    # Check if button exists and is visible
                    if page.locator(selector).is_visible():
                        print(f"   🔘 Found wake-up button with selector: {selector}")
                        
                        # Click the button
                        page.locator(selector).click()
                        print(f"   ✅ Clicked wake-up button!")
                        
                        # Wait for app to start waking up
                        page.wait_for_timeout(5000)
                        
                        result['success'] = True
                        result['status'] = 'woken_up'
                        result['message'] = 'Successfully clicked wake-up button'
                        button_found = True
                        break
                        
                except PlaywrightTimeoutError:
                    continue
                except Exception as e:
                    print(f"   ⚠️  Error with selector {selector}: {e}")
                    continue
            
            if not button_found:
                # Check if app is already running by looking for Streamlit-specific elements
                streamlit_indicators = [
                    "[data-testid='stApp']",
                    ".main .block-container",
                    "[data-testid='stSidebar']",
                    ".stApp"
                ]
                
                app_running = False
                for indicator in streamlit_indicators:
                    try:
                        if page.locator(indicator).is_visible():
                            app_running = True
                            break
                    except:
                        continue
                
                if app_running:
                    print(f"   ✅ App is already running!")
                    result['success'] = True
                    result['status'] = 'already_awake'
                    result['message'] = 'App is already running'
                else:
                    print(f"   ❓ No wake-up button found and app status unclear")
                    result['status'] = 'unclear'
                    result['message'] = 'No wake-up button found, app status unclear'
            
        except PlaywrightTimeoutError:
            result['message'] = f'Timeout after {self.timeout/1000} seconds'
            print(f"   ⏰ Timeout: {result['message']}")
            
        except Exception as e:
            result['message'] = f'Error: {str(e)}'
            print(f"   ❌ Error: {e}")
            
        finally:
            if page:
                page.close()
            
            print(f"   ⏱️  Response time: {result['response_time']:.2f}s")
            print(f"   📊 Result: {result['status']}")
        
        return result
    
    def wake_up_multiple_apps(self, urls: List[str]) -> List[dict]:
        """
        Wake up multiple Streamlit apps.
        
        Args:
            urls: List of Streamlit app URLs
            
        Returns:
            List[dict]: Results for each app
        """
        results = []
        
        print(f"🚀 Starting wake-up process for {len(urls)} apps...")
        print("=" * 60)
        
        for i, url in enumerate(urls, 1):
            print(f"\n[{i}/{len(urls)}] Processing: {url}")
            result = self.wake_up_app(url)
            results.append(result)
            
            # Small delay between apps to be respectful
            if i < len(urls):
                time.sleep(2)
        
        return results
    
    def print_summary(self, results: List[dict]):
        """Print a summary of all wake-up attempts."""
        print("\n" + "=" * 60)
        print("📋 WAKE-UP SUMMARY")
        print("=" * 60)
        
        successful = 0
        already_awake = 0
        failed = 0
        
        for result in results:
            status_emoji = {
                'woken_up': '✅',
                'already_awake': '🟢', 
                'unclear': '❓',
                'unknown': '❌'
            }.get(result['status'], '❌')
            
            print(f"{status_emoji} {result['url']}")
            print(f"   Status: {result['status']}")
            print(f"   Message: {result['message']}")
            print(f"   Response time: {result['response_time']:.2f}s")
            
            if result['status'] == 'woken_up':
                successful += 1
            elif result['status'] == 'already_awake':
                already_awake += 1
            else:
                failed += 1
        
        print(f"\n📊 STATISTICS:")
        print(f"   🎯 Successfully woken up: {successful}")
        print(f"   🟢 Already awake: {already_awake}")
        print(f"   ❌ Failed/Unclear: {failed}")
        print(f"   📱 Total apps: {len(results)}")
        
        success_rate = ((successful + already_awake) / len(results)) * 100 if results else 0
        print(f"   📈 Success rate: {success_rate:.1f}%")


def main():
    """Main function to handle command line execution."""
    parser = argparse.ArgumentParser(
        description='Wake up Streamlit apps using Playwright',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python wake_streamlit.py https://app1.streamlit.app/
  python wake_streamlit.py https://app1.streamlit.app/ https://app2.streamlit.app/
  python wake_streamlit.py --visible https://app1.streamlit.app/
        """
    )
    
    parser.add_argument(
        'urls',
        nargs='+',
        help='URLs of Streamlit apps to wake up'
    )
    
    parser.add_argument(
        '--visible',
        action='store_true',
        help='Run browser in visible mode (not headless)'
    )
    
    parser.add_argument(
        '--timeout',
        type=int,
        default=30,
        help='Timeout in seconds for page operations (default: 30)'
    )
    
    args = parser.parse_args()
    
    # Validate URLs
    valid_urls = []
    for url in args.urls:
        if not url.startswith(('http://', 'https://')):
            print(f"⚠️  Warning: Adding https:// to {url}")
            url = 'https://' + url
        valid_urls.append(url)
    
    # Run the wake-up process
    try:
        with StreamlitWakeUp(headless=not args.visible, timeout=args.timeout * 1000) as waker:
            results = waker.wake_up_multiple_apps(valid_urls)
            waker.print_summary(results)
            
            # Exit with appropriate code
            failed_count = sum(1 for r in results if r['status'] not in ['woken_up', 'already_awake'])
            sys.exit(0 if failed_count == 0 else 1)
            
    except KeyboardInterrupt:
        print("\n🛑 Process interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n💥 Fatal error: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()